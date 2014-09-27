package main
 
import (
	"bytes"
	"encoding/hex"
	//"testing"
	"time"
	"sort"
	"fmt"
	"log"
	"io/ioutil"
	"path/filepath"
 
	"github.com/conformal/btcjson"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcrpcclient"

	"github.com/FactomProject/FactomCode/notaryapi"
	
)

//var client *btcrpcclient.Client
var currentAddr btcutil.Address
var fee btcutil.Amount
var wif *btcutil.WIF

// ByAmount defines the methods needed to satisify sort.Interface to
// sort a slice of Utxos by their amount.
type ByAmount []btcjson.ListUnspentResult

func (u ByAmount) Len() int           { return len(u) }
func (u ByAmount) Less(i, j int) bool { return u[i].Amount < u[j].Amount }
func (u ByAmount) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }


func SendRawTransactionToBTC(hash []byte) (*btcwire.ShaHash, error) {
	
	//addrStr := "muhXX7mXoMZUBvGLCgfjuoY2n2mziYETYC"
	
	//if err := initRPCClient(); err != nil {
		//return fmt.Errorf("cannot init rpc client: %s", err)
	//}
	//defer shutdown(client)
	
	//if err := initWallet(addrStr); err != nil {
		//return fmt.Errorf("cannot init wallet: %s", err)
	//}
	
	msgtx, err := createRawTransaction(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot create Raw Transaction: %s", err)
	}
	
	shaHash, err := sendRawTransaction(msgtx)
	if err != nil {
		return nil, fmt.Errorf("cannot send Raw Transaction: %s", err)
	}
	
	return shaHash, nil
}


func initWallet(addrStr string) error {
	
	fee, _ = btcutil.NewAmount(0.0001)
	
	err := client.WalletPassphrase("lindasilva", int64(2))
	if err != nil {
		return fmt.Errorf("cannot unlock wallet with passphrase: %s", err)
	}	

	currentAddr, err = btcutil.DecodeAddress(addrStr, &btcnet.TestNet3Params)
	if err != nil {
		return fmt.Errorf("cannot decode address: %s", err)
	}

	wif, err = client.DumpPrivKey(currentAddr) 
	if err != nil { 
		return fmt.Errorf("cannot get WIF: %s", err)
	}

	return nil
	
}


func createRawTransaction(hash []byte) (*btcwire.MsgTx, error) {
	
	msgtx := btcwire.NewMsgTx()

	minconf := 0	// 1
	maxconf := 999999
	
	addrs := []btcutil.Address{currentAddr}
		
	unspent, err := client.ListUnspentMinMaxAddresses(minconf, maxconf, addrs)
	if err != nil {
		return nil, fmt.Errorf("cannot ListUnspentMinMaxAddresses: %s", err)
	}
	fmt.Printf("unspent, len=%d", len(unspent))

	// Sort eligible inputs, as unspent expects these to be sorted
	// by amount in reverse order.
	sort.Sort(sort.Reverse(ByAmount(unspent)))
	
	inputs, btcin, err := selectInputs(unspent, minconf)
	if err != nil {
		return nil, fmt.Errorf("cannot selectInputs: %s", err)
	}
	fmt.Println("selectedInputs, len=%d", len(inputs))
	
	change := btcin - fee
	if err = addTxOuts(msgtx, change, hash); err != nil {
		return nil, fmt.Errorf("cannot addTxOuts: %s", err)
	}

	if err = addTxIn(msgtx, inputs); err != nil {
		return nil, fmt.Errorf("cannot addTxIn: %s", err)
	}

	if err = validateMsgTx(msgtx, inputs); err != nil {
		return nil, fmt.Errorf("cannot validateMsgTx: %s", err)
	}

	return msgtx, nil
}



// For every unspent output given, add a new input to the given MsgTx. Only P2PKH outputs are
// supported at this point.
func addTxIn(msgtx *btcwire.MsgTx, outputs []btcjson.ListUnspentResult) error {
	
	for _, output := range outputs {
		fmt.Printf("unspentResult: %#v", output)
		prevTxHash, err := btcwire.NewShaHashFromStr(output.TxId)
		if err != nil {
			return fmt.Errorf("cannot get sha hash from str: %s", err)
		}
		
		outPoint := btcwire.NewOutPoint(prevTxHash, output.Vout)
		msgtx.AddTxIn(btcwire.NewTxIn(outPoint, nil))
	}

	for i, output := range outputs {
	 
		subscript, err := hex.DecodeString(output.ScriptPubKey)
		if err != nil {
			return fmt.Errorf("cannot decode scriptPubKey: %s", err)
		}
		
		fmt.Println("subscript ", string(subscript))
	 
		sigScript, err := btcscript.SignatureScript(msgtx, i, subscript,
			btcscript.SigHashAll, wif.PrivKey.ToECDSA(), true)
		if err != nil {
			return fmt.Errorf("cannot create scriptSig: %s", err)
		}
		msgtx.TxIn[i].SignatureScript = sigScript
		
		fmt.Println("sigScript ", string(sigScript))
		
	}
	return nil
}

func addTxOuts(msgtx *btcwire.MsgTx, change btcutil.Amount, hash []byte) error {
 
 	header := []byte{0x46, 0x61, 0x63, 0x74, 0x6f, 0x6d, 0x21, 0x21}	// Factom!!
	//hash := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
	//			   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
	//			   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
	//			   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	hash = append(header, hash...)

	
	builder := btcscript.NewScriptBuilder()
	builder.AddOp(btcscript.OP_RETURN)
	builder.AddData(hash)
	opReturn := builder.Script()
	msgtx.AddTxOut(btcwire.NewTxOut(0, opReturn))

	// Check if there are leftover unspent outputs, and return coins back to
	// a new address we own.
	if change > 0 {

		// Spend change.
		pkScript, err := btcscript.PayToAddrScript(currentAddr)
		if err != nil {
			return fmt.Errorf("cannot create txout script: %s", err)
		}
		//btcscript.JSONToAmount(jsonAmount float64) (int64)
		msgtx.AddTxOut(btcwire.NewTxOut(int64(change), pkScript))
	}
	return nil
}



// selectInputs selects the minimum number possible of unspent
// outputs to use to create a new transaction that spends amt satoshis.
// btcout is the total number of satoshis which would be spent by the
// combination of all selected previous outputs.  err will equal
// ErrInsufficientFunds if there are not enough unspent outputs to spend amt
// amt.
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
		panic(err)
	}
	
	txRawResult, err := client.DecodeRawTransaction(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot Decode Raw Transaction: %s", err)
	}
	fmt.Println("txRawResult: ", txRawResult)
	
	shaHash, err := client.SendRawTransaction(msgtx, false)
	if err != nil {
		return nil, fmt.Errorf("cannot send Raw Transaction: %s", err)
	}
	fmt.Println("btc txHash: ", shaHash)	// new tx hash
	
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
		     fmt.Println("OnAccountBalance, account=", account, ", balance=", balance.String, ", confirmed=", confirmed)
	    },
		OnBlockConnected: func(hash *btcwire.ShaHash, height int32) {
			fmt.Println("OnBlockConnected")
			//go newBlock(hash, height)	// no need
		},
	}
	
	// Connect to local btcwallet RPC server using websockets.
	certHomeDir := btcutil.AppDataDir("btcwallet", false)
	certs, err := ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return fmt.Errorf("cannot read rpc.cert file: %s", err)
	}
	connCfg := &btcrpcclient.ConnConfig{
		Host:         "localhost:18332",
		Endpoint:     "ws",
		User:         "testuser",
		Pass:         "notarychain",
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

//
// newBlock hashes the current LastHash into the Bitcoin Blockchain 5 minutes after
// the previous block. (If a block is signed quicker than 5 minutes, then the second
// block is ignored.)
//
//func newBlock(hash *btcwire.ShaHash, height int32) {

func newBlock() {

	if waiting {
		return
	}
	waiting = true
	 
	//log.Printf("Block connected: %v (%d)", hash, height)

	time.Sleep(time.Minute * 1)		//5)

	// acquire the last block
	//   no one else will change the blocks array, so we don't need to lock to safely acquire
	block := blocks[len(blocks)-1]

	// wait until it's full
	for len(block.EBEntries) < 3 {
		time.Sleep(time.Minute)
		log.Print("waiting for entries... have ",len(block.EBEntries))
	}

	// add a new block for new entries to be added to
	blockMutex.Lock()
	newblock, _ := notaryapi.CreateBlock(block, 10)
	blocks = append(blocks, newblock)
	blockMutex.Unlock()

	blkhash, _ := notaryapi.CreateHash(block)
	hashdata := blkhash.Bytes
	
	fmt.Printf("hashdata.len=%d", len(hashdata))
	
	txHash, err := SendRawTransactionToBTC(hashdata)
	if err != nil {
		log.Fatalf("cannot init rpc client: %s", err)
	}
    log.Print("Recorded ", hashdata, " in transaction hash:\n",txHash)
    
	waiting = false    //??
}

