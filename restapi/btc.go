package main

import (
	//"github.com/conformal/btcrpcclient"
	"encoding/hex"
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/conformal/btcjson"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcrpcclient"
	"log"
	"strings"
	"time"
	"bytes"
	
	"github.com/conformal/btcec"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcscript"	
	
	"encoding/binary"
	"math/rand"	
	"io/ioutil"
	"path/filepath"	
	"fmt"
	"strconv"
) 

// Enodes up to 30 bytes into a 33 byte Bitcoin public key.
// Returns the public key.  The format is as follows:
// 1      byte   02  (per Bitcoin spec)
// 1      byte   len (Number of bytes encoded, between 1 and 63)
// len    bytes  encoded data
// 30-len bytes  random data
// fudge  byte   changed to put the value on the eliptical curve
//
func Encode(hash []byte) ([]byte, error) {
	length := len(hash)
	if length == 0 || length > 30 {
		return nil, errors.New("Encode can only handle 1 to 30 bytes")
	}
	var b []byte = make([]byte, 0, 33)
	b = append(b, byte(2), byte(length))

	b = append(b, hash...)

	if length < 30 {
		data := make([]byte, 30-length)
		b = append(b, data...)
	}

	b = append(b, byte(0))

	for i := 0; i < 256; i++ {
		b[len(b)-1] = byte(i)
		adr2 := hex.EncodeToString(b)
		_, e := btcutil.DecodeAddress(adr2, activeNet.Params)
		if e == nil {
			return b, nil
		}
	}

	return b, errors.New("Couldn't fix the address")
}

//
// Faithfully extracts upto 30 bytes encoded into the given bitcoin address
func Decode(addr []byte) []byte {
	length := int(addr[1])
	data := addr[2 : length+2]
	return data
}

func newBalance(account string, balance btcutil.Amount, confirmed bool) {
	sconf := "unconfirmed"
	if confirmed {
		sconf = "confirmed"
	}
	log.Printf("New %s balance for account %s: %v", sconf, account, balance)
}

// Compute the balance for the currentAddr, and the list of its unspent
// outputs
func computeBalance() (cAmount btcutil.Amount, cList []btcjson.TransactionInput, err error) {

	// Get the list of unspent transaction outputs (utxos) that the
	// connected wallet has at least one private key for.

	unspent, e := client.ListUnspent()
	if e != nil {
		err = e
		return
	}

	// This is going to be our map of addresses to all unspent outputs
	var outputs = make(map[string][]btcjson.ListUnspentResult)

	for _, input := range unspent {
		l, n := outputs[input.Address] // Get the list of
		if !n {
			l = make([]btcjson.ListUnspentResult, 1)
			l[0] = input
			outputs[input.Address] = l
		} else {
			outputs[input.Address] = append(l, input)
		}
	}

	for index, unspentList := range outputs {
		if strings.EqualFold(index, (*currentAddr).EncodeAddress()) {
			cAmount = btcutil.Amount(0)
			for i := range unspentList {
				cAmount += btcutil.Amount(unspentList[i].Amount * float64(100000000))
			}
			cList = make([]btcjson.TransactionInput, len(unspentList), len(unspentList))
			for i, u := range unspentList {
				v := new(btcjson.TransactionInput)
				v.Txid = u.TxId
				v.Vout = u.Vout
				cList[i] = *v
			}
		}
	}
	return
}

/**
 * Record the hash into the Bitcoin Block Chain
**/
func recordHash(lastHash []byte) (txhash *btcwire.ShaHash) {

	b0, _ := Encode(lastHash[:16])
	adr0 := hex.EncodeToString(b0)
	b1, _ := Encode(lastHash[16:])
	adr1 := hex.EncodeToString(b1)

	address := make([]string, 2, 2)
	decodeAdr := make([]btcutil.Address, 2, 2)
	results := make([]*btcjson.ValidateAddressResult, 2, 2)

	address[0] = adr0 //external compressed public key
	address[1] = adr1 //external compressed public key

	for i := 0; i < len(address); i++ {
		var err error

		decodeAdr[i], err = btcutil.DecodeAddress(address[i], activeNet.Params)
		if err != nil {
			return
		}

		results[i], err = client.ValidateAddress(decodeAdr[i])
		if err != nil {

			return
		}

	}

	addressSlice := append(make([]btcutil.Address, 0, 3), *currentAddr, decodeAdr[0], decodeAdr[1])

	multiAddr, e := client.AddMultisigAddress(1, addressSlice, "")

	if e != nil {
		return
	}

	amount, unspent, err0 := computeBalance()
	fee, err1 := btcutil.NewAmount(.0005)
	change := amount - 5430 - fee
	send, err2 := btcutil.NewAmount(.00005430)

	if err0 != nil || err1 != nil || err2 != nil {
		log.Print("Reported Error: ", err1, err2)
		return
	}
	/*
		log.Print("Amount at the address:  ", amount)
		log.Print("Change after the trans: ", change)
		log.Print("Amount to send:         ", send)
		log.Print("Send+Change+fee:        ", send+change+fee)
		log.Print("unspent: ",unspent)
	*/

	adrs := make(map[btcutil.Address]btcutil.Amount)
	adrs[multiAddr] = send
	// dest, _ := btcutil.DecodeAddress("mnyUYs1SJFQEKSLFZoGUsUTk8mZbbt37Ge", activeNet.Params);
	// adrs[dest] = send
	adrs[*currentAddr] = change
	rawTrans, err3 := client.CreateRawTransaction(unspent, adrs)

	if err3 != nil {
		log.Print("raw trans create failed", err3)
		return
	}

	signedTrans, inputsSigned, err4 := client.SignRawTransaction(rawTrans)

	if err4 != nil {
		log.Print("Failed to sign transaction ", err4)
		return
	}

	if !inputsSigned {
		log.Print("Inputs are not signed;  Is your wallet unlocked? ")
		return
	}

	var err error

	txhash, err = client.SendRawTransaction(signedTrans, false)

	if err != nil {
		log.Print("Transaction submission failed", err)
		return
	}

	return

}

var waiting bool = false

//
// newBlock hashes the current LastHash into the Bitcoin Blockchain 5 minutes after
// the previous block. (If a block is signed quicker than 5 minutes, then the second
// block is ignored.)
//
/*func newBlock(hash *btcwire.ShaHash, height int32) {

	if waiting {
		return
	}
	waiting = true
	 
	log.Printf("Block connected: %v (%d)", hash, height)

	time.Sleep(time.Minute * 5)

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
	
	db.ProcessEBlockBatche(blkhash, block) //??
	
	//rework needed ??
	SendTransactionToBTC(hashdata)
	//txhash := recordHash(hashdata)
    //log.Print("Recorded ",hashdata," in transaction hash:\n",txhash)
    
	waiting = false    //??
}
*/

func newEntryBlock(chain *notaryapi.Chain) (block *notaryapi.Block){



	// acquire the last block
	//   no one else will change the blocks array, so we don't need to lock to safely acquire
	block = chain.Blocks[len(chain.Blocks)-1]

	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	block.EBEntries, _ = db.FetchEntriesFromQueue(chain.ChainID, &binaryTimestamp)

 	if len(block.EBEntries) < 1{
 		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
 		return nil
 	}
 

	blkhash, _ := notaryapi.CreateHash(block)
//	hashdata := blkhash.Bytes
	
	db.ProcessEBlockBatche(blkhash, block) //??

	// add a new block for new entries to be added to
	chain.BlockMutex.Lock()
	newblock, _ := notaryapi.CreateBlock(chain, block, 10)
	newblock.Chain = chain
	
	
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	
	//rework needed ??
//	SendTransactionToBTC(hashdata)
	//txhash := recordHash(hashdata)
    //log.Print("Recorded ",hashdata," in transaction hash:\n",txhash)
    
	waiting = false    //??
	
	log.Println("block" + strconv.FormatUint(block.BlockID, 10) +" created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))	
	return block
}

func newFactomBlock(chain *notaryapi.FChain) {



	// acquire the last block
	//   no one else will change the blocks array, so we don't need to lock to safely acquire
	block := chain.Blocks[len(chain.Blocks)-1]

	//binaryTimestamp := make([]byte, 8)
	//binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	//block.EBEntries, _ = db.FetchEntriesFromQueue(chain.ChainID, &binaryTimestamp)

 	if len(block.FBEntries) < 1{
 		//log.Println("No Factom block created for chain ... because no new entry is found.")
 		return
 	}
 

	//blkhash, _ := notaryapi.CreateHash(block)
//	hashdata := blkhash.Bytes
	
	//db.ProcessFBlockBatche(blkhash, block) //??

	// add a new block for new entries to be added to
	chain.BlockMutex.Lock()
	newblock, _ := notaryapi.CreateFBlock(chain, block, 10)
	newblock.Sealed = false
	
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	
	//rework needed ??
//	SendTransactionToBTC(hashdata)
	//txhash := recordHash(hashdata)
    //log.Print("Recorded ",hashdata," in transaction hash:\n",txhash)
    
	waiting = false    //??
	block.Sealed = true
	log.Println("block" + strconv.FormatUint(block.BlockID, 10) +" created for factom chain: "  + notaryapi.EncodeChainID(chain.ChainID))	
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


//------------------------------------------

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
 
func SendTransactionToBTC(hash []byte) {
 
	rand.Seed(23234241)
 
	// Create from key pairs.
	fromPrivKey, fromPubKey := newKeyPair()
 
	// fromAddr is mx6nrkeysWh5k5nm8foSJPGBdMU8GjYtHC.
	fromAddrPubKey, err := btcutil.NewAddressPubKey(
		fromPubKey.SerializeCompressed(), &btcnet.TestNet3Params)
	if err != nil {
		log.Println(err)
	}
	
	// This might not be needed.
	fromAddrPubKeyHash := fromAddrPubKey.AddressPubKeyHash()
	
	log.Println("fromAddrPubKeyHash ", fromAddrPubKeyHash)
 
	// Create to key pairs.
	_, toPubKey := newKeyPair()
 
	// toAddr is mvgvChMs6Wc6HgDUEwRE8d8YEzfmG5qhEY.
	toAddrPubKey, err := btcutil.NewAddressPubKey(
		toPubKey.SerializeCompressed(), &btcnet.TestNet3Params)
	if err != nil {
		panic(err)
	}



	// This might not be needed.
	toAddrPubKeyHash := toAddrPubKey.AddressPubKeyHash()
	
	log.Println("toAddrPubKeyHash ", toAddrPubKeyHash)
 
	// Create our transaction.
	// Our only input is from output 1 of the folliwng Testnet3 transaction:
	// a3246d39853ceda87e517758d4fc1beef2f52d2551a2809dc20a23c7e43e05e0.
	tx := btcwire.NewMsgTx()
 

 
	fromAddrPkScript, err := btcscript.PayToAddrScript(fromAddrPubKeyHash)
	if err != nil {
		panic(err)
	}
	tx.AddTxOut(btcwire.NewTxOut(50000, fromAddrPkScript))	// return change 70000 - 20000 = 50000
	
 
 
 
 	header := []byte{0x46, 0x61, 0x63, 0x74, 0x6f, 0x6d, 0x21, 0x21}	// Factom!!
/*	hash := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 
				   0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	*/
 	hash = append(header, hash...)

	
	builder := btcscript.NewScriptBuilder()
	builder.AddOp(btcscript.OP_RETURN)
	builder.AddData(hash)
	opReturn := builder.Script()
	tx.AddTxOut(btcwire.NewTxOut(0, opReturn))
	
	disasm1, err := btcscript.DisasmString(opReturn)
	if err != nil {
		log.Println(err)
		return
	}

	disasm2, err2 := btcscript.DisasmString(fromAddrPkScript)
	if err2 != nil {
		log.Println(err2)
		return
	}
	
	
	
	log.Println("Op_Return Hex:      %x", opReturn)
	log.Println("Op_Return:          ", disasm1)	
	log.Println("Change Hex:         %x", fromAddrPkScript)
	log.Println("Change Disassembly: ", disasm2)

 
 
	// Create txIn.
	//prevOutTxHashStr := "ee082d4de5d03478ff19405ca8307b10dc110b3ff27a4f7b1bff1f3a7731c8a9" //prev tx hash for prevOutTXHashStr
	//prevOutTxHashStr := "3b9b8c5c6f4aa308f518ea7b6243da7139bc1d4fac3e71f89b2b48a766b96383"
	prevOutTxHashStr := "5aa17c2430e685d77ae16cab3d3e9d7218c49e00e844155d1062c1443f46034b"
	
	prevOutTxHash, err := btcwire.NewShaHashFromStr(prevOutTxHashStr)
	if err != nil {
		panic(err)
	}
	
	log.Println("prevOutTxHash ", prevOutTxHash)
 
	prevOut := btcwire.NewOutPoint(prevOutTxHash, 1)
 
	txIn := btcwire.NewTxIn(prevOut, nil)
	tx.AddTxIn(txIn)
 
	// 76     19         14 b5e8473dab40a4da81aa13a6c2f7c7da9b8b2458 88             ac  	// scriptPubKey of out 
	// OP_DUP OP_HASH160 <pubKeyHash>                                OP_EQUALVERIFY OP_CHECKSIG
	subscriptHex := "76a914b5e8473dab40a4da81aa13a6c2f7c7da9b8b245888ac"
 
	subscript, err := hex.DecodeString(subscriptHex)
	if err != nil {
		panic(err)
	}
	
	log.Println("subscript ", string(subscript))
 
	sigScript, err := btcscript.SignatureScript(tx, 0, subscript,
		btcscript.SigHashAll, fromPrivKey.ToECDSA(), true)
	if err != nil {
		panic(err)
	}
	tx.TxIn[0].SignatureScript = sigScript
	
	log.Println("sigScript ", string(sigScript))
 
	buf := bytes.Buffer{}
	if err := tx.Serialize(&buf); err != nil {
		panic(err)
	}
 
	txHex := hex.EncodeToString(buf.Bytes())
	log.Println("tx.Hex ", txHex)
	
	
	// init btcrpcclient.Client
	
	client := initRPCClient()
	
	txRawResult, err := client.DecodeRawTransaction(buf.Bytes())
	if err != nil {log.Println(err.Error())}
	log.Println("txRawResult: ", txRawResult)
	
	shaHash, err := client.SendRawTransaction(tx, false)
	if err != nil {log.Println(err.Error())}
	log.Println("shaHash: ", shaHash)	// new tx hash
	
	
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



