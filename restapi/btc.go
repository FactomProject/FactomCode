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

 
	"github.com/FactomProject/btcjson"
	"github.com/FactomProject/btcd/chaincfg"
	"github.com/FactomProject/btcd/txscript"
	"github.com/FactomProject/btcutil"
	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/btcws"
	"github.com/FactomProject/btcrpcclient"

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


// failedMerkles stores to-be-written-to-btc FactomBlock Hash
// after it failed for maxTrials attempt 
var failedMerkles []*notaryapi.Hash

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


func writeToBTC(bytes []byte) (*wire.ShaHash, error) {	
	for attempts := 0; attempts < maxTrials; attempts++ {
		txHash, err := SendRawTransactionToBTC(bytes)
		if err != nil {
			log.Printf("Attempt %d to send raw tx to BTC failed: %s\n", attempts, err)
			time.Sleep(time.Duration(attempts*20) * time.Second)
			continue
		}
		return txHash, nil
	}
	return nil, fmt.Errorf("Fail to write hash %s to BTC: %s", bytes)
}


func SendRawTransactionToBTC(hash []byte) (*wire.ShaHash, error) {
	b := balances[0]
	i := copy(balances, balances[1:])
	balances[i] = b
	//balances[0:] = balances[1:]
	//balances[len(balances) - 1] = b
	
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
	failedMerkles = make([]*notaryapi.Hash, 0, 100)
	
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
		addr, err := btcutil.DecodeAddress(b.unspentResult.Address, &chaincfg.TestNet3Params)
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


func createRawTransaction(b balance, hash []byte) (*wire.MsgTx, error) {
	
	msgtx := wire.NewMsgTx()
	
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

func addTxIn(msgtx *wire.MsgTx, b balance) error {
	
	output := b.unspentResult
//	fmt.Printf("unspentResult: %#v\n", output)
	prevTxHash, err := wire.NewShaHashFromStr(output.TxId)
	if err != nil {
		return fmt.Errorf("cannot get sha hash from str: %s", err)
	}
	
	outPoint := wire.NewOutPoint(prevTxHash, output.Vout)
	msgtx.AddTxIn(wire.NewTxIn(outPoint, nil))

	// OnRedeemingTx
	err = dclient.NotifySpent([]*wire.OutPoint {outPoint})
	if err != nil {
		fmt.Println("NotifySpent err: ", err.Error())
	}

	subscript, err := hex.DecodeString(output.ScriptPubKey)
	if err != nil {
		return fmt.Errorf("cannot decode scriptPubKey: %s", err)
	}
 
	sigScript, err := txscript.SignatureScript(msgtx, 0, subscript,
		txscript.SigHashAll, b.wif.PrivKey, true)	//.ToECDSA(), true)
	if err != nil {
		return fmt.Errorf("cannot create scriptSig: %s", err)
	}
	msgtx.TxIn[0].SignatureScript = sigScript
	
	return nil
}


func addTxOuts(msgtx *wire.MsgTx, b balance, hash []byte) error {
 
 	header := []byte{0x46, 0x61, 0x63, 0x74, 0x6f, 0x6d, 0x21, 0x21}	// Factom!!
	hash = append(header, hash...)
	
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData(hash)
	opReturn, err := builder.Script()
	msgtx.AddTxOut(wire.NewTxOut(0, opReturn))
	
	if err != nil {
		fmt.Printf("ScriptBuilder error: %v\n", err)
	}

	amount, _ := btcutil.NewAmount(b.unspentResult.Amount)
	change := amount - fee

	// Check if there are leftover unspent outputs, and return coins back to
	// a new address we own.
	if change > 0 {

		// Spend change.
		pkScript, err := txscript.PayToAddrScript(b.address)
		if err != nil {
			return fmt.Errorf("cannot create txout script: %s", err)
		}
		msgtx.AddTxOut(wire.NewTxOut(int64(change), pkScript))
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


func validateMsgTx(msgtx *wire.MsgTx, inputs []btcjson.ListUnspentResult) error {
	flags := txscript.ScriptCanonicalSignatures | txscript.ScriptStrictMultiSig
	bip16 := time.Now().After(txscript.Bip16Activation)
	if bip16 {
		flags |= txscript.ScriptBip16
	}
	for i, txin := range msgtx.TxIn {
	 
		subscript, err := hex.DecodeString(inputs[i].ScriptPubKey)
		if err != nil {
			return fmt.Errorf("cannot decode scriptPubKey: %s", err)
		}

		engine, err := txscript.NewScript(
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


func sendRawTransaction(msgtx *wire.MsgTx) (*wire.ShaHash, error) {

	buf := bytes.Buffer{}
	buf.Grow(msgtx.SerializeSize())
	if err := msgtx.BtcEncode(&buf, wire.ProtocolVersion); err != nil {
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

	ntfnHandlers := btcrpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
		     //go newBalance(account, balance, confirmed)
		     //fmt.Println("wclient: OnAccountBalance, account=", account, ", balance=", 
		     //	balance.ToUnit(btcutil.AmountBTC), ", confirmed=", confirmed)
	    },

		OnWalletLockState: func(locked bool) {
			fmt.Println("wclient: OnWalletLockState, locked=", locked)
		},

		OnUnknownNotification: func(method string, params []json.RawMessage) {
			//fmt.Println("wclient: OnUnknownNotification: method=", method, "\nparams[0]=", 
			//	string(params[0]), "\nparam[1]=", string(params[1]))
		},
	}
	
	return ntfnHandlers
}



func createBtcdNotificationHandlers() btcrpcclient.NotificationHandlers {

	ntfnHandlers := btcrpcclient.NotificationHandlers{

		OnBlockConnected: func(hash *wire.ShaHash, height int32) {
			//fmt.Println("dclient: OnBlockConnected: hash=", hash, ", height=", height)
			//go newBlock(hash, height)	// no need
		},

		OnRecvTx: func(transaction *btcutil.Tx, details *btcws.BlockDetails) {
			//fmt.Printf("dclient: OnRecvTx: details=%#v\n", details)
			//fmt.Printf("dclient: OnRecvTx: tx=%#v,  tx.Sha=%#v, tx.index=%d\n", 
				//transaction, transaction.Sha().String(), transaction.Index())
		},

		OnRedeemingTx: func(transaction *btcutil.Tx, details *btcws.BlockDetails) {
			fmt.Printf("dclient: OnRedeemingTx: details=%#v\n", details)
			fmt.Printf("dclient: OnRedeemingTx: tx.Sha=%#v,  tx.index=%d\n", 
				transaction.Sha().String(), transaction.Index())
			
			if details != nil {
				// do not block OnRedeemingTx callback
				go saveDBBatch(transaction, details)
			}
		},
	}
	 
	return ntfnHandlers
}


func saveDBBatch(transaction *btcutil.Tx, details *btcws.BlockDetails) {
	fmt.Println("In saveDBBatch, len(dbBatches.batches)=", len(dbBatches.batches))
	var i int
	var found bool
	for i = 0; i < len(dbBatches.batches); i++ {
		fmt.Printf("i=%d, dbBatch=%#v\n", i, dbBatches.batches[i])
		
		if dbBatches.batches[i].BTCTxHash != nil && 
			bytes.Compare(dbBatches.batches[i].BTCTxHash.Bytes, transaction.Sha().Bytes()) == 0 {
			
			dbBatches.batches[i].BTCTxOffset = details.Index
			dbBatches.batches[i].BTCBlockHeight = details.Height
			
			txHash, _ := wire.NewShaHashFromStr(details.Hash)
			dbBatches.batches[i].BTCBlockHash = toHash(txHash)
			
			found = true
			break
		}
	}
	fmt.Println("In saveFBBatch, found=", found)

	if found {
		fmt.Printf("found in dbBatches: i=%d, len=%d, DELETE dbBatch%#v\n",i , len(dbBatches.batches), dbBatches.batches[i])
		
		//delete dbBatches.batches[i]
		dbBatch := dbBatches.batches[i]
		dbBatches.batchMutex.Lock()
		dbBatches.batches = append(dbBatches.batches[:i], dbBatches.batches[i+1:] ...)
		/*
		copy(dbBatches.batches[i:], dbBatches.batches[i+1:])
		dbBatches.batches[len(dbBatches.batches) - 1] = nil
		dbBatches.batches = dbBatches.batches[:len(dbBatches.batches) - 1]
		*/
		dbBatches.batchMutex.Unlock()
		
		// Update db with DBBatch
		db.InsertDBBatch(dbBatch)
		ExportDataFromDbToFile()						

		fmt.Println("found in dbBatches: after deletion, len=", len(dbBatches.batches))
	}
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


func newEntryBlock(chain *notaryapi.EChain) (*notaryapi.EBlock){

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

 	if len(block.EBEntries) < 1{
 		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
 		return nil
 	}

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	blkhash, _ := notaryapi.CreateHash(block)
	log.Println("blkhash:%v", blkhash.Bytes)
	block.IsSealed = true	
	chain.NextBlockID++	
	newblock, _ := notaryapi.CreateBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	block.EBHash = blkhash	
	
	// Create the Entry Block Merkle Root for FB Entry
	hashes := make([]*notaryapi.Hash, 0, len(block.EBEntries)+1)
	for _, entry := range block.EBEntries {
		data, _ := entry.MarshalBinary()
		hashes = append (hashes, notaryapi.Sha(data))
	}
	binaryEBHeader, _ := block.Header.MarshalBinary()
	hashes = append (hashes, notaryapi.Sha(binaryEBHeader))	
	merkle := notaryapi.BuildMerkleTreeStore(hashes)
	block.MerkleRoot = merkle[len(merkle) - 1]	// MerkleRoot is not marshalized in Entry Block
	fmt.Println("block.MerkleRoot:%v", block.MerkleRoot.String())
	    
    //Store the block in db
	db.ProcessEBlockBatch(block)	 
	log.Println("EntryBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) +" created for chain: "  + chain.ChainID.String())	
	
	return block
}

func newEntryCreditBlock(chain *notaryapi.CChain) (*notaryapi.CBlock){

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

 	if len(block.CBEntries) < 1{
 		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
 		return nil
 	}

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.EntryCount = uint32(len(block.CBEntries))		
	blkhash, _ := notaryapi.CreateHash(block)
	log.Println("blkhash:%v", blkhash.Bytes)
	block.IsSealed = true	
	chain.NextBlockID++	
	newblock, _ := notaryapi.CreateCBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	block.CBHash = blkhash	
	    
    //Store the block in db
	db.ProcessCBlockBatch(block)	   
	log.Println("EntryCreditBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) +" created for chain: "  + chain.ChainID.String())	
	
	return block
}


func newDirectoryBlock(chain *notaryapi.DChain) *notaryapi.DBlock {

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

 	if len(block.DBEntries) < 1{
 		//log.Println("No Directory block created for chain ... because no new entry is found.")
 		return nil
 	} 
	
	// Create the block add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.EntryCount = uint32(len(block.DBEntries))	
	// Calculate Merkle Root for FBlock and store it in header
	if block.Header.MerkleRoot == nil {
		block.Header.MerkleRoot = block.CalculateMerkleRoot()
		fmt.Println("block.Header.MerkleRoot:%v", block.Header.MerkleRoot.String())
	}	
	blkhash, _ := notaryapi.CreateHash(block)
	block.IsSealed = true	
	chain.NextBlockID++
	newblock, _ := notaryapi.CreateDBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()

	//Store the block in db
	block.DBHash = blkhash
	db.ProcessDBlockBatch(block)  	

	log.Println("DirectoryBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) +" created for directory block chain: "  + chain.ChainID.String())

	//update FBBlock with FBHash & FBlockID

	//block.FBlockID = block.Header.BlockID
	
	//Export all db records associated w/ this new factom block
	ExportDbToFile(blkhash)
	
	return block
}


func saveDBBatchMerkleRoottoBTC(dbBatch *notaryapi.DBBatch) {
	fmt.Println("in saveFBBatchMerkleRoottoBTC: len(dbBatch.DBlocks)=", len(dbBatch.DBlocks))
	
	//calculate batch merkle root
	hashes := make([]*notaryapi.Hash, 0, len(dbBatch.DBlocks))
	for i:=0; i<len(dbBatch.DBlocks); i++ {
		fmt.Printf("i=%d, merkle root: %s\n", i, dbBatch.DBlocks[i].Header.MerkleRoot.String())
		hashes = append(hashes, dbBatch.DBlocks[i].Header.MerkleRoot)
	}	
	merkle := notaryapi.BuildMerkleTreeStore(hashes)
	merkleRoot := merkle[len(merkle) - 1]
	dbBatch.FBBatchMerkleRoot = merkleRoot

	txHash, err := writeToBTC(merkleRoot.Bytes)		
	if err != nil {
		failedMerkles = append(failedMerkles, merkleRoot)
		fmt.Println("failed to record ", merkleRoot.Bytes, " to BTC: ", err.Error())
	}

	//convert btc tx hash to factom hash, and update dbBatch
	dbBatch.BTCTxHash = toHash(txHash)

    fmt.Print("Recorded FBBatch merkle root in BTC tx hash:\n",txHash, "\nconverted hash: ", dbBatch.BTCTxHash.String(), "\n")

}


func toHash(txHash *wire.ShaHash) *notaryapi.Hash {
	h := new (notaryapi.Hash)
	h.SetBytes(txHash.Bytes())
	return h
}

