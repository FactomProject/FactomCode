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
func newBlock(hash *btcwire.ShaHash, height int32) {

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
	for len(block.Entries) < 5 {
		time.Sleep(time.Minute)
		log.Print("waiting for entries... have ",len(block.Entries))
	}

	// add a new block for new entries to be added to
	blockMutex.Lock()
	newblock, _ := notaryapi.CreateBlock(block, 10)
	blocks = append(blocks, newblock)
	blockMutex.Unlock()

	blkhash, _ := notaryapi.CreateHash(block)
	hashdata := blkhash.Bytes

	txhash := recordHash(hashdata)
    log.Print("Recorded ",hashdata," in transaction hash:\n",txhash)
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
