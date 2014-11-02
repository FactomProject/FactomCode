package main
 
import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
//	"sort"
	"fmt"
	"log"
	"io/ioutil"
	"path/filepath"
 
	"github.com/conformal/btcjson"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcws"
	"github.com/conformal/btcrpcclient"

	"github.com/FactomProject/FactomCode/notaryapi"
	
)

var fee btcutil.Amount
//var wif *btcutil.WIF
var balances []balance
var blkHashFailed []*notaryapi.Hash
const maxTries = 3

// the spentResult is the one showing up in ListSpent()
// but is already spent in blockexploer
// it's a bug in btcwallet
var spentResult = btcjson.ListUnspentResult {
	TxId:"1a3450d99659d5b704d89c26d56082a0f13ba2a275fdd9ffc0ec4f42c88fe857", 
	Vout:0xb, 
	Address:"mvwnVraAK1VKRcbPgrSAVZ6E5hkqbwRxCy", 
	Account:"", 
	ScriptPubKey:"76a914a93c1baaaeae1b30688edab5e927fb2bfc794cae88ac", 
	RedeemScript:"", 
	Amount:1, 
	Confirmations:28247,
}

// ByAmount defines the methods needed to satisify sort.Interface to
// sort a slice of Utxos by their amount.
type ByAmount []btcjson.ListUnspentResult

func (u ByAmount) Len() int           { return len(u) }
func (u ByAmount) Less(i, j int) bool { return u[i].Amount < u[j].Amount }
func (u ByAmount) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

type balance struct {
	unspentResult 	btcjson.ListUnspentResult
	address			btcutil.Address
	wif 			*btcutil.WIF
}


func writeToBTC(hash *notaryapi.Hash) (*btcwire.ShaHash, error) {	
	for attempts := 0; attempts < maxTries; attempts++ {
		txHash, err := SendRawTransactionToBTC(hash.Bytes)
		if err != nil {
			log.Printf("Attempt %d to send raw tx to BTC failed: %s\n", attempts, err)
			time.Sleep(time.Duration(attempts*20) * time.Second)
			continue
		}
		return txHash, nil
	}
	blkHashFailed = append(blkHashFailed, hash)
	return nil, fmt.Errorf("Fail to write hash %s to BTC: %s", hash)
}


func SendRawTransactionToBTC(hash []byte) (*btcwire.ShaHash, error) {
	b := balances[0]
	i := copy(balances, balances[1:])
	balances[i] = b
	
	msgtx, err := createRawTransaction(b, hash)
	if err != nil {
		return nil, fmt.Errorf("cannot create Raw Transaction: %s", err)
	}
	
	shaHash, err := sendRawTransaction(msgtx)
	if err != nil {
		return nil, fmt.Errorf("cannot send Raw Transaction: %s", err)
	}
	return shaHash, nil
}


func getAddress(b *balance) error {
	addr, err := btcutil.DecodeAddress(b.unspentResult.Address, &btcnet.TestNet3Params)
	if err != nil {
		return fmt.Errorf("cannot decode address: %s", err)
	}
	b.address = addr

	wif, err := client.DumpPrivKey(addr)
	if err != nil {	
		return fmt.Errorf("cannot get WIF: %s", err)
	}
	b.wif = wif

	return nil
}


func initWallet() error {
	fee, _ = btcutil.NewAmount(btcTransFee)
	blkHashFailed = make([]*notaryapi.Hash, 0, 100)
	
	err := client.WalletPassphrase(walletPassphrase, int64(2))
	if err != nil {
		return fmt.Errorf("cannot unlock wallet with passphrase: %s", err)
	}

	unspentResults, err := client.ListUnspent()	//minConf=1
	if err != nil {
		return fmt.Errorf("cannot list unspent. %s", err)
	}
	//fmt.Println("unspentResults.len=", len(unspentResults))

	balances = make([]balance, 0, 100)
	if len(unspentResults) > 0 {
		var i int
		for _, b := range unspentResults {
		
			//bypass the bug in btcwallet where one unconfirmed spent 
			//is listed in unspent result
			if b.Amount > float64(0.1) && !compareUnspentResult(spentResult, b) {
				//fmt.Println(i, "  ", b.Amount)
				balances = append(balances, balance{unspentResult : b})
				i++
			}			
		}
	}
	fmt.Println("balances.len=", len(balances))

	for i, b := range balances {
		addr, err := btcutil.DecodeAddress(b.unspentResult.Address, &btcnet.TestNet3Params)
		if err != nil {
			return fmt.Errorf("cannot decode address: %s", err)
		}
		balances[i].address = addr

		wif, err := client.DumpPrivKey(addr)
		if err != nil {	
			return fmt.Errorf("cannot get WIF: %s", err)
		}
		balances[i].wif = wif
		
		//fmt.Println(balances[i])
	}	

	return nil
}


func compareUnspentResult(a, b btcjson.ListUnspentResult) bool {
	if a.TxId == b.TxId && a.Vout == b.Vout && a.Amount == b.Amount && a.Address == b.Address {
		return true
	} else {
		return false
	}
}


func createRawTransaction(b balance, hash []byte) (*btcwire.MsgTx, error) {
	
	msgtx := btcwire.NewMsgTx()
	
	if err := addTxOuts(msgtx, b, hash); err != nil {
		return nil, fmt.Errorf("cannot addTxOuts: %s", err)
	}

	if err := addTxIn(msgtx, b); err != nil {
		return nil, fmt.Errorf("cannot addTxIn: %s", err)
	}

	if err := validateMsgTx(msgtx, []btcjson.ListUnspentResult{b.unspentResult}); err != nil {
		return nil, fmt.Errorf("cannot validateMsgTx: %s", err)
	}

	return msgtx, nil
}

func addTxIn(msgtx *btcwire.MsgTx, b balance) error {
	
	output := b.unspentResult
	fmt.Printf("unspentResult: %#v\n", output)
	prevTxHash, err := btcwire.NewShaHashFromStr(output.TxId)
	if err != nil {
		return fmt.Errorf("cannot get sha hash from str: %s", err)
	}
	
	outPoint := btcwire.NewOutPoint(prevTxHash, output.Vout)
	msgtx.AddTxIn(btcwire.NewTxIn(outPoint, nil))

	subscript, err := hex.DecodeString(output.ScriptPubKey)
	if err != nil {
		return fmt.Errorf("cannot decode scriptPubKey: %s", err)
	}		
	//fmt.Println("\nsubscript ", string(subscript))
 
	sigScript, err := btcscript.SignatureScript(msgtx, 0, subscript,
		btcscript.SigHashAll, b.wif.PrivKey.ToECDSA(), true)
	if err != nil {
		return fmt.Errorf("cannot create scriptSig: %s", err)
	}
	msgtx.TxIn[0].SignatureScript = sigScript
	
	//fmt.Println("sigScript ", string(sigScript))
	return nil
}


func addTxOuts(msgtx *btcwire.MsgTx, b balance, hash []byte) error {
 
 	header := []byte{0x46, 0x61, 0x63, 0x74, 0x6f, 0x6d, 0x21, 0x21}	// Factom!!
	hash = append(header, hash...)
	
	builder := btcscript.NewScriptBuilder()
	builder.AddOp(btcscript.OP_RETURN)
	builder.AddData(hash)
	opReturn := builder.Script()
	msgtx.AddTxOut(btcwire.NewTxOut(0, opReturn))

	amount, _ := btcutil.NewAmount(b.unspentResult.Amount)
	change := amount - fee

	// Check if there are leftover unspent outputs, and return coins back to
	// a new address we own.
	if change > 0 {

		// Spend change.
		pkScript, err := btcscript.PayToAddrScript(b.address)
		if err != nil {
			return fmt.Errorf("cannot create txout script: %s", err)
		}
		msgtx.AddTxOut(btcwire.NewTxOut(int64(change), pkScript))
	}
	return nil
}


func validateMsgTx(msgtx *btcwire.MsgTx, inputs []btcjson.ListUnspentResult) error {
	flags := btcscript.ScriptCanonicalSignatures | btcscript.ScriptStrictMultiSig
	bip16 := time.Now().After(btcscript.Bip16Activation)
	if bip16 {
		flags |= btcscript.ScriptBip16
	}
	for i, txin := range msgtx.TxIn {
	 
		subscript, err := hex.DecodeString(inputs[i].ScriptPubKey)
		if err != nil {
			return fmt.Errorf("cannot decode scriptPubKey: %s", err)
		}

		engine, err := btcscript.NewScript(
			txin.SignatureScript, subscript, i, msgtx, flags)
		if err != nil {
			return fmt.Errorf("cannot create script engine: %s", err)
		}
		if err = engine.Execute(); err != nil {
			return fmt.Errorf("cannot validate transaction: %s", err)
		}
	}
	return nil
}


func sendRawTransaction(msgtx *btcwire.MsgTx) (*btcwire.ShaHash, error) {

	buf := bytes.Buffer{}
	buf.Grow(msgtx.SerializeSize())
	if err := msgtx.BtcEncode(&buf, btcwire.ProtocolVersion); err != nil {
		// Hitting OOM by growing or writing to a bytes.Buffer already
		// panics, and all returned errors are unexpected.
		//panic(err) //?? should we have retry logic?
		return nil, err
	}
	
	shaHash, err := client.SendRawTransaction(msgtx, false)
	if err != nil {
		return nil, fmt.Errorf("failed in rpcclient.SendRawTransaction: %s", err)
	}
	fmt.Println("btc txHash returned: ", shaHash)	// new tx hash
	
	return shaHash, nil
}



func initRPCClient() error {
	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the btcrpcclient
	// NotificationHandlers type for more details about each handler.
	ntfnHandlers := btcrpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
		     //go newBalance(account, balance, confirmed)
		     fmt.Println("OnAccountBalance, account=", account, ", balance=", balance.ToUnit(btcutil.AmountBTC), ", confirmed=", confirmed)
	    },
		OnBlockConnected: func(hash *btcwire.ShaHash, height int32) {
			fmt.Println("OnBlockConnected")
			//go newBlock(hash, height)	// no need
		},
		// OnClientConnected is invoked when the client connects or reconnects
		// to the RPC server.  This callback is run async with the rest of the
		// notification handlers, and is safe for blocking client requests.
		OnClientConnected: func() {
			fmt.Println("OnClientConnected")
		},

		// OnBlockConnected is invoked when a block is connected to the longest
		// (best) chain.  It will only be invoked if a preceding call to
		// NotifyBlocks has been made to register for the notification and the
		// function is non-nil.
		//OnBlockConnected func(hash *btcwire.ShaHash, height int32)
		
		// OnBlockDisconnected is invoked when a block is disconnected from the
		// longest (best) chain.  It will only be invoked if a preceding call to
		// NotifyBlocks has been made to register for the notification and the
		// function is non-nil.
		OnBlockDisconnected: func(hash *btcwire.ShaHash, height int32) {
			fmt.Println("OnBlockDisconnected: hash=", hash, ", height=", height)
		},

		// OnRecvTx is invoked when a transaction that receives funds to a
		// registered address is received into the memory pool and also
		// connected to the longest (best) chain.  It will only be invoked if a
		// preceding call to NotifyReceived, Rescan, or RescanEndHeight has been
		// made to register for the notification and the function is non-nil.
		OnRecvTx: func(transaction *btcutil.Tx, details *btcws.BlockDetails) {
			fmt.Printf("OnRecvTx: tx=%#v", transaction, ", details=%#v", details, "\n")
		},

		// OnRedeemingTx is invoked when a transaction that spends a registered
		// outpoint is received into the memory pool and also connected to the
		// longest (best) chain.  It will only be invoked if a preceding call to
		// NotifySpent, Rescan, or RescanEndHeight has been made to register for
		// the notification and the function is non-nil.
		//
		// NOTE: The NotifyReceived will automatically register notifications
		// for the outpoints that are now "owned" as a result of receiving
		// funds to the registered addresses.  This means it is possible for
		// this to invoked indirectly as the result of a NotifyReceived call.
		OnRedeemingTx: func(transaction *btcutil.Tx, details *btcws.BlockDetails) {
			fmt.Printf("OnRedeemingTx: tx=%#v", transaction, ", details=%#v", details, "\n")
		},

		// OnRescanFinished is invoked after a rescan finishes due to a previous
		// call to Rescan or RescanEndHeight.  Finished rescans should be
		// signaled on this notification, rather than relying on the return
		// result of a rescan request, due to how btcd may send various rescan
		// notifications after the rescan request has already returned.
		OnRescanFinished: func(hash *btcwire.ShaHash, height int32, blkTime time.Time) {
			fmt.Println("OnRescanFinished: hash=", hash, ", height=", height, ", blkTime=", blkTime)
		},

		// OnRescanProgress is invoked periodically when a rescan is underway.
		// It will only be invoked if a preceding call to Rescan or
		// RescanEndHeight has been made and the function is non-nil.
		OnRescanProgress: func(hash *btcwire.ShaHash, height int32, blkTime time.Time) {
			fmt.Println("OnRescanProgress: hash=", hash, ", height=", height, ", blkTime=", blkTime)
		},

		// OnTxAccepted is invoked when a transaction is accepted into the
		// memory pool.  It will only be invoked if a preceding call to
		// NotifyNewTransactions with the verbose flag set to false has been
		// made to register for the notification and the function is non-nil.
		OnTxAccepted: func(hash *btcwire.ShaHash, amount btcutil.Amount) {
			fmt.Println("OnTxAccepted: hash=", hash, ", amount=", amount)
		},

		// OnTxAccepted is invoked when a transaction is accepted into the
		// memory pool.  It will only be invoked if a preceding call to
		// NotifyNewTransactions with the verbose flag set to true has been
		// made to register for the notification and the function is non-nil.
		OnTxAcceptedVerbose: func(txDetails *btcjson.TxRawResult) {
			fmt.Printf("OnTxAcceptedVerbose: txDetails=%#v", txDetails, "\n")
		},

		// OnBtcdConnected is invoked when a wallet connects or disconnects from
		// btcd.
		//
		// This will only be available when client is connected to a wallet
		// server such as btcwallet.
		OnBtcdConnected: func(connected bool) { 
			fmt.Println("OnBtcdConnected, connected=", connected)
		},

		// OnAccountBalance is invoked with account balance updates.
		//
		// This will only be available when speaking to a wallet server
		// such as btcwallet.
		//OnAccountBalance func(account string, balance btcutil.Amount, confirmed bool)

		// OnWalletLockState is invoked when a wallet is locked or unlocked.
		//
		// This will only be available when client is connected to a wallet
		// server such as btcwallet.
		OnWalletLockState: func(locked bool) {
			fmt.Println("OnWalletLockState, locked=", locked)
		},

		// OnUnknownNotification is invoked when an unrecognized notification
		// is received.  This typically means the notification handling code
		// for this package needs to be updated for a new notification type or
		// the caller is using a custom notification this package does not know
		// about.
		OnUnknownNotification: func(method string, params []json.RawMessage) {
			var msgTx btcwire.MsgTx
			_ = json.Unmarshal(params[0], &msgTx)	
			fmt.Printf("OnUnknownNotification: method=", method, ", param=%#v", msgTx, "\n")
		},
	}
	 
	// Connect to local btcwallet RPC server using websockets.
	certHomeDir := btcutil.AppDataDir(certHomePath, false)
	certs, err := ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return fmt.Errorf("cannot read rpc.cert file: %s", err)
	}
	connCfg := &btcrpcclient.ConnConfig{
		Host:         rpcClientHost,
		Endpoint:     rpcClientEndpoint,
		User:         rpcClientUser,
		Pass:         rpcClientPass,
		Certificates: certs,
	}
	
	client, err = btcrpcclient.New(connCfg, &ntfnHandlers)	
	if err != nil {
		return fmt.Errorf("cannot create rpc client: %s", err)
	}
	
	return nil
}


func shutdown(client *btcrpcclient.Client) {
	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	log.Println("Client shutdown in 2 seconds...")
	time.AfterFunc(time.Second*2, func() {
		log.Println("Going down...")
		client.Shutdown()
	})
	defer log.Println("Shutdown done!")
	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	client.WaitForShutdown()
}


var waiting bool = false


func newEntryBlock(chain *notaryapi.Chain) (*notaryapi.Block, *notaryapi.Hash){

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

 	if len(block.EBEntries) < 1{
 		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
 		return nil, nil
 	}

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	blkhash, _ := notaryapi.CreateHash(block)
	block.IsSealed = true	
	chain.NextBlockID++	
	newblock, _ := notaryapi.CreateBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
    
    //Store the block in db
	db.ProcessEBlockBatch(blkhash, block)	
	log.Println("block" + strconv.FormatUint(block.Header.BlockID, 10) +" created for chain: "  + chain.ChainID.String())	
	
	return block, blkhash
}


func newFactomBlock(chain *notaryapi.FChain) {

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

 	if len(block.FBEntries) < 1{
 		//log.Println("No Factom block created for chain ... because no new entry is found.")
 		return
 	} 
	
	// Create the block add a new block for new coming entries
	chain.BlockMutex.Lock()
	blkhash, _ := notaryapi.CreateHash(block)
	block.IsSealed = true	
	chain.NextBlockID++
	newblock, _ := notaryapi.CreateFBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessFBlockBatch(blkhash, block) 	
	//need to add a FB process queue in db??	
	log.Println("block" + strconv.FormatUint(block.Header.BlockID, 10) +" created for factom chain: "  + notaryapi.EncodeBinary(chain.ChainID))
	
	//Send transaction to BTC network
	txHash, _ := writeToBTC(blkhash)		
	
	// Create a FBInfo and insert it into db
	fbInfo := new (notaryapi.FBInfo)
	fbInfo.FBHash = blkhash
	fbInfo.FBlockID = block.Header.BlockID

	if (txHash != nil) {
		btcTxHash := new (notaryapi.Hash)
		btcTxHash.Bytes = txHash.Bytes()
		fbInfo.BTCTxHash = btcTxHash
	}
	
	db.InsertFBInfo(blkhash, fbInfo)
	
	// Export all db records associated w/ this new factom block
	ExportDbToFile(blkhash)
	
	if (txHash != nil) {
	    log.Print("Recorded ", blkhash.Bytes, " in BTC transaction hash:\n",txHash)
	} else {
		log.Println("failed to record ", blkhash.Bytes, " to BTC" )
	}
    
    
}
