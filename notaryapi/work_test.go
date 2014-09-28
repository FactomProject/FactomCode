package notaryapi
 
import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"testing"
	"fmt"
	"log"
	"io/ioutil"
	"path/filepath"
	"time"
	
 
	"github.com/conformal/btcec"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcrpcclient"
	//"github.com/conformal/btcjson"
	
)
 
// newKeyPair should only ever be used for testing as it uses a
// pseudo random number generator that will generate a predictable
// series of key pairs. Use crypto/rand for MainNet.
func newKeyPair() (*btcec.PrivateKey, *btcec.PublicKey) {
	pkBytes := make([]byte, 32)
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(pkBytes[i*4:i*4+4], rand.Uint32())
	}
 
	return btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)
}
 
func TestSendTransaction(t *testing.T) {
 
	rand.Seed(23234241)
 
	// Create from key pairs.
	fromPrivKey, fromPubKey := newKeyPair()
 
	// fromAddr is mx6nrkeysWh5k5nm8foSJPGBdMU8GjYtHC.
	fromAddrPubKey, err := btcutil.NewAddressPubKey(
		fromPubKey.SerializeCompressed(), &btcnet.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	
	// This might not be needed.
	fromAddrPubKeyHash := fromAddrPubKey.AddressPubKeyHash()
	
	t.Log("fromAddrPubKeyHash ", fromAddrPubKeyHash)
 
	// Create to key pairs.
	_, toPubKey := newKeyPair()
 
	// toAddr is mvgvChMs6Wc6HgDUEwRE8d8YEzfmG5qhEY.
	toAddrPubKey, err := btcutil.NewAddressPubKey(
		toPubKey.SerializeCompressed(), &btcnet.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}



	// This might not be needed.
	toAddrPubKeyHash := toAddrPubKey.AddressPubKeyHash()
	
	t.Log("toAddrPubKeyHash ", toAddrPubKeyHash)
 
	// Create our transaction.
	// Our only input is from output 1 of the folliwng Testnet3 transaction:
	// a3246d39853ceda87e517758d4fc1beef2f52d2551a2809dc20a23c7e43e05e0.
	tx := btcwire.NewMsgTx()
 

 
	fromAddrPkScript, err := btcscript.PayToAddrScript(fromAddrPubKeyHash)
	if err != nil {
		t.Fatal(err)
	}
	tx.AddTxOut(btcwire.NewTxOut(50000, fromAddrPkScript))	// return change 70000 - 20000 = 50000
	
 
 
 
 	header := []byte{0x46, 0x61, 0x63, 0x74, 0x6f, 0x6d, 0x21, 0x21}	// Factom!!
	hash := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	hash = append(header, hash...)

	
	builder := btcscript.NewScriptBuilder()
	builder.AddOp(btcscript.OP_RETURN)
	builder.AddData(hash)
	opReturn := builder.Script()
	tx.AddTxOut(btcwire.NewTxOut(0, opReturn))
	
	disasm1, err := btcscript.DisasmString(opReturn)
	if err != nil {
		t.Error(err)
		return
	}

	disasm2, err2 := btcscript.DisasmString(fromAddrPkScript)
	if err2 != nil {
		t.Error(err2)
		return
	}
	
	
	
	t.Logf("Op_Return Hex:      %x", opReturn)
	t.Logf("Op_Return:          ", disasm1)	
	t.Logf("Change Hex:         %x", fromAddrPkScript)
	t.Logf("Change Disassembly: ", disasm2)

 
 
	// Create txIn.
	//prevOutTxHashStr := "ee082d4de5d03478ff19405ca8307b10dc110b3ff27a4f7b1bff1f3a7731c8a9" //prev tx hash for prevOutTXHashStr
	prevOutTxHashStr := "3b9b8c5c6f4aa308f518ea7b6243da7139bc1d4fac3e71f89b2b48a766b96383"
	prevOutTxHash, err := btcwire.NewShaHashFromStr(prevOutTxHashStr)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Log("prevOutTxHash ", prevOutTxHash)
 
	prevOut := btcwire.NewOutPoint(prevOutTxHash, 1)
 
	txIn := btcwire.NewTxIn(prevOut, nil)
	tx.AddTxIn(txIn)
 
	// 76     19         14 b5e8473dab40a4da81aa13a6c2f7c7da9b8b2458 88             ac  	// scriptPubKey of out 
	// OP_DUP OP_HASH160 <pubKeyHash>                                OP_EQUALVERIFY OP_CHECKSIG
	subscriptHex := "76a914b5e8473dab40a4da81aa13a6c2f7c7da9b8b245888ac"
 
	subscript, err := hex.DecodeString(subscriptHex)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Log("subscript ", string(subscript))
 
	sigScript, err := btcscript.SignatureScript(tx, 0, subscript,
		btcscript.SigHashAll, fromPrivKey.ToECDSA(), true)
	if err != nil {
		t.Fatal(err)
	}
	tx.TxIn[0].SignatureScript = sigScript
	
	t.Log("sigScript ", string(sigScript))
 
	buf := bytes.Buffer{}
	if err := tx.Serialize(&buf); err != nil {
		t.Fatal(err)
	}
 
	txHex := hex.EncodeToString(buf.Bytes())
	t.Log("tx.Hex ", txHex)
	
	
	// init btcrpcclient.Client
	
	client := initRPCClient()
	
	txRawResult, err := client.DecodeRawTransaction(buf.Bytes())
	if err != nil {t.Error(err.Error())}
	t.Log("txRawResult: ", txRawResult)
	
	shaHash, err := client.SendRawTransaction(tx, false)
	if err != nil {t.Error(err.Error())}
	t.Log("shaHash: ", shaHash)	// new tx hash
	
	
	defer shutdown(client)
	
}



func initRPCClient() (c *btcrpcclient.Client) {

	//cadr, err := btcutil.DecodeAddress("mjx5q1BwAfgtJ1UPFoRXucphaM9k1dtzbf", activeNet.Params)

	//currentAddr = &cadr

	// Only override the handlers for notifications you care about.
	// Also note most of the handlers will only be called if you register
	// for notifications.  See the documentation of the btcrpcclient
	// NotificationHandlers type for more details about each handler.
	ntfnHandlers := btcrpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
		     //go newBalance(account, balance, confirmed)
		     fmt.Println("OnAccountBalance")
	    },

		//OnBlockConnected: func(hash *btcwire.ShaHash, height int32) {
			//go newBlock(hash, height)
		//},
	}
	
	// Connect to local btcwallet RPC server using websockets.
	certHomeDir := btcutil.AppDataDir("btcwallet", false)
	certs, err := ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
		return nil
	}
	connCfg := &btcrpcclient.ConnConfig{
		Host:         "localhost:18332",
		Endpoint:     "ws",
		User:         "testuser",
		Pass:         "notarychain",
		Certificates: certs,
	}
	
	client, err := btcrpcclient.New(connCfg, &ntfnHandlers)
	
	if err != nil {
		log.Fatal(err)
		return nil;
	}
	
	return client
}


func shutdown(client *btcrpcclient.Client) {
	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	log.Println("Client shutdown in 2 seconds...")
	time.AfterFunc(time.Second*2, func() {
		/* =============> */ log.Println("Going down...")
		client.Shutdown()
	})
	defer /* =============> */ log.Println("Shutdown done!")
	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	client.WaitForShutdown()
}

