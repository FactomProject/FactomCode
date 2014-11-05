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


// fee is paid to miner for tx written into btc
var fee btcutil.Amount

// balances store unspent balance & address & its WIF
var balances []balance
type balance struct {
	unspentResult 	btcjson.ListUnspentResult
	address			btcutil.Address
	wif 			*btcutil.WIF
}

// blockDetailsMap stores txHash & blockdetails(block hash, height and offset)
// for those tx being confirmed in btc blockchain
var blockDetailsMap map[string]*btcws.BlockDetails

// blkHashFailed stores to-be-written-to-btc FactomBlock Hash
// after it failed for maxTrials attempt 
var blkHashFailed []*notaryapi.Hash

// maxTrials is the max attempts to writeToBTC
const maxTrials = 3

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


func writeToBTC(hash *notaryapi.Hash) (*btcwire.ShaHash, error) {	
	for attempts := 0; attempts < maxTrials; attempts++ {
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


func unlockWallet(timeoutSecs int64) error {
	err := wclient.WalletPassphrase(walletPassphrase, int64(timeoutSecs))
	if err != nil {
		return fmt.Errorf("cannot unlock wallet with passphrase: %s", err)
	}
	return nil
}


func initWallet() error {
	fee, _ = btcutil.NewAmount(btcTransFee)
	blkHashFailed = make([]*notaryapi.Hash, 0, 100)
	blockDetailsMap = make(map[string]*btcws.BlockDetails)
	
	err := unlockWallet(int64(1))
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	unspentResults, err := wclient.ListUnspent()	//minConf=1
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
//	fmt.Println("balances.len=", len(balances))

	for i, b := range balances {
		addr, err := btcutil.DecodeAddress(b.unspentResult.Address, &btcnet.TestNet3Params)
		if err != nil {
			return fmt.Errorf("cannot decode address: %s", err)
		}
		balances[i].address = addr

		wif, err := wclient.DumpPrivKey(addr)
		if err != nil {	
			return fmt.Errorf("cannot get WIF: %s", err)
		}
		balances[i].wif = wif
		
//		fmt.Println(balances[i])
	}
	
//	registerNotifications()

	time.Sleep(1 * time.Second)

	return nil
}


func registerNotifications() {
	// OnBlockConnected or OnBlockDisconnected
	err := dclient.NotifyBlocks()
	if err != nil {
		fmt.Println("NotifyBlocks err: ", err.Error())
	}
	
	// OnTxAccepted: not useful since it covers all addresses
	err = dclient.NotifyNewTransactions(false)	//verbose is false
	if err != nil {
		fmt.Println("NotifyNewTransactions err: ", err.Error())
	}
	
	// OnRecvTx 
	addresses := make([]btcutil.Address, 0, 30)
	for _, a := range balances {
		addresses = append(addresses, a.address)
	}
	err = dclient.NotifyReceived(addresses)
	if err != nil {
		fmt.Println("NotifyReceived err: ", err.Error())
	}

	// OnRedeemingTx
	//err := dclient.NotifySpent(outpoints)
	
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
//	fmt.Printf("unspentResult: %#v\n", output)
	prevTxHash, err := btcwire.NewShaHashFromStr(output.TxId)
	if err != nil {
		return fmt.Errorf("cannot get sha hash from str: %s", err)
	}
	
	outPoint := btcwire.NewOutPoint(prevTxHash, output.Vout)
	msgtx.AddTxIn(btcwire.NewTxIn(outPoint, nil))

	// OnRedeemingTx
	err = dclient.NotifySpent([]*btcwire.OutPoint {outPoint})
	if err != nil {
		fmt.Println("NotifySpent err: ", err.Error())
	}

	subscript, err := hex.DecodeString(output.ScriptPubKey)
	if err != nil {
		return fmt.Errorf("cannot decode scriptPubKey: %s", err)
	}
 
	sigScript, err := btcscript.SignatureScript(msgtx, 0, subscript,
		btcscript.SigHashAll, b.wif.PrivKey.ToECDSA(), true)
	if err != nil {
		return fmt.Errorf("cannot create scriptSig: %s", err)
	}
	msgtx.TxIn[0].SignatureScript = sigScript
	
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


func selectInputs(eligible []btcjson.ListUnspentResult, minconf int) (selected []btcjson.ListUnspentResult, out btcutil.Amount, err error) {
	// Iterate throguh eligible transactions, appending to outputs and
	// increasing out.  This is finished when out is greater than the
	// requested amt to spend.
	selected = make([]btcjson.ListUnspentResult, 0, len(eligible))
	for _, e := range eligible {
		amount, err := btcutil.NewAmount(e.Amount)
		if err != nil {
			fmt.Println("err in creating NewAmount")
			continue
		}
		selected = append(selected, e)
		out += amount
		if out >= fee {
			return selected, out, nil
		}
	}
	if out < fee {
		return nil, 0, fmt.Errorf("insufficient funds: transaction requires %v fee, but only %v spendable", fee, out)		 
	}

	return selected, out, nil
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
	
	// use rpc client for btcd here for better callback info
	// this should not require wallet to be unlocked
	shaHash, err := dclient.SendRawTransaction(msgtx, false)
	if err != nil {
		return nil, fmt.Errorf("failed in rpcclient.SendRawTransaction: %s", err)
	}
	fmt.Println("btc txHash returned: ", shaHash)	// new tx hash
	
	return shaHash, nil
}


func createBtcwalletNotificationHandlers() btcrpcclient.NotificationHandlers {
	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the btcrpcclient
	// NotificationHandlers type for more details about each handler.
	ntfnHandlers := btcrpcclient.NotificationHandlers{
	
		// OnAccountBalance is invoked with account balance updates.
		//
		// This will only be available when speaking to a wallet server
		// such as btcwallet.
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
		     //go newBalance(account, balance, confirmed)
		     //fmt.Println("wclient: OnAccountBalance, account=", account, ", balance=", balance.ToUnit(btcutil.AmountBTC), ", confirmed=", confirmed)
	    },

		// OnWalletLockState is invoked when a wallet is locked or unlocked.
		//
		// This will only be available when client is connected to a wallet
		// server such as btcwallet.
		OnWalletLockState: func(locked bool) {
			fmt.Println("wclient: OnWalletLockState, locked=", locked)
		},

		// OnUnknownNotification is invoked when an unrecognized notification
		// is received.  This typically means the notification handling code
		// for this package needs to be updated for a new notification type or
		// the caller is using a custom notification this package does not know
		// about.
		OnUnknownNotification: func(method string, params []json.RawMessage) {
			//fmt.Println("wclient: OnUnknownNotification: method=", method, "\nparams[0]=", string(params[0]), "\nparam[1]=", string(params[1]))
		},
	}
	
	return ntfnHandlers
}



func createBtcdNotificationHandlers() btcrpcclient.NotificationHandlers {
	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the btcrpcclient
	// NotificationHandlers type for more details about each handler.
	ntfnHandlers := btcrpcclient.NotificationHandlers{

		// OnBlockConnected is invoked when a block is connected to the longest
		// (best) chain.  It will only be invoked if a preceding call to
		// NotifyBlocks has been made to register for the notification and the
		// function is non-nil.
		OnBlockConnected: func(hash *btcwire.ShaHash, height int32) {
			//fmt.Println("dclient: OnBlockConnected: hash=", hash, ", height=", height)
			//go newBlock(hash, height)	// no need
		},

		// OnRecvTx is invoked when a transaction that receives funds to a
		// registered address is received into the memory pool and also
		// connected to the longest (best) chain.  It will only be invoked if a
		// preceding call to NotifyReceived, Rescan, or RescanEndHeight has been
		// made to register for the notification and the function is non-nil.
		OnRecvTx: func(transaction *btcutil.Tx, details *btcws.BlockDetails) {
			//fmt.Printf("dclient: OnRecvTx: details=%#v\n", details)
			//fmt.Printf("dclient: OnRecvTx: tx=%#v,  tx.Sha=%#v, tx.index=%d\n", 
				//transaction, transaction.Sha().String(), transaction.Index())
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
			blockDetailsMap[transaction.Sha().String()] = details
			fmt.Printf("dclient: OnRedeemingTx: details=%#v\n", details)
			fmt.Printf("dclient: OnRedeemingTx: tx.Sha=%#v,  tx.index=%d\n", 
				transaction.Sha().String(), transaction.Index())
		},
	}
	
	return ntfnHandlers
}


func initRPCClient() error {
	 
	// Connect to local btcwallet RPC server using websockets.
	ntfnHandlers := createBtcwalletNotificationHandlers()
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
	wclient, err = btcrpcclient.New(connCfg, &ntfnHandlers)	
	if err != nil {
		return fmt.Errorf("cannot create rpc client for btcwallet: %s", err)
	}
	
	// Connect to local btcd RPC server using websockets.
	dntfnHandlers := createBtcdNotificationHandlers()
	certHomeDir = btcutil.AppDataDir(certHomePathBtcd, false)
	certs, err = ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return fmt.Errorf("cannot read rpc.cert file for btcd rpc server: %s", err)
	}	
	dconnCfg := &btcrpcclient.ConnConfig{
		Host:         rpcBtcdHost,	
		Endpoint:     rpcClientEndpoint,
		User:         rpcClientUser,
		Pass:         rpcClientPass,
		Certificates: certs,
	}	
	dclient, err = btcrpcclient.New(dconnCfg, &dntfnHandlers)	
	if err != nil {
		return fmt.Errorf("cannot create rpc client for btcd: %s", err)
	}
	
	return nil
}


func shutdown() {
	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	log.Println("Client shutdown in 2 seconds...")
	time.AfterFunc(time.Second*2, func() {
		log.Println("Going down...")
		wclient.Shutdown()
		dclient.Shutdown()
	})
	defer log.Println("Shutdown done!")
	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	wclient.WaitForShutdown()
	dclient.WaitForShutdown()
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
