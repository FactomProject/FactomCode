//  Goxed because it goes away too long... Maybe this should have a
//  short test and skip.

package anchor

import (
	"fmt"
	//"log"
	//"reflect"
	"testing"
	//"time"

	//"github.com/FactomProject/FactomCode/common"
	//"github.com/FactomProject/FactomCode/database"
	//"github.com/FactomProject/FactomCode/database/ldb"
	//"github.com/FactomProject/FactomCode/util"
	//"github.com/btcsuitereleases/btcd/btcjson"
	//"github.com/btcsuitereleases/btcd/wire"
	//"github.com/btcsuitereleases/btcutil"
	//"github.com/davecgh/go-spew/spew"
)

func TestInitRPCClient(t *testing.T) {
	fmt.Println("* see details in ~/.factom/logs/factom-d.log")
	err := InitRPCClient()
	if err == nil {
		fmt.Println("successfully created rpc client for both btcd and btcwallet.")
	} else {
		fmt.Println(err.Error())
	}
}

/*
// maxTrials is the max attempts to writeToBTC
const maxTrials = 3

// the spentResult is the one showing up in ListSpent()
// but is already spent in blockexploer
// it's a bug in btcwallet
var spentResult = btcjson.ListUnspentResult{
	TxID:          "1a3450d99659d5b704d89c26d56082a0f13ba2a275fdd9ffc0ec4f42c88fe857",
	Vout:          0xb,
	Address:       "mvwnVraAK1VKRcbPgrSAVZ6E5hkqbwRxCy",
	Account:       "",
	ScriptPubKey:  "76a914a93c1baaaeae1b30688edab5e927fb2bfc794cae88ac",
	RedeemScript:  "",
	Amount:        1,
	Confirmations: 28247,
}

func compareUnspentResult(a, b btcjson.ListUnspentResult) bool {
	if a.TxID == b.TxID && a.Vout == b.Vout && a.Amount == b.Amount && a.Address == b.Address {
		return true
	}
	return false
}

func TestPrependBlockHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Log("testing...")

	s1 := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	const h1 uint64 = 0x123456789ABC // temp block height

	_, err := prependBlockHeight(h1, s1)
	if nil != err {
		t.Errorf("error 1!")
	}

	const h2 uint64 = 0
	_, err = prependBlockHeight(h2, s1)
	if nil != err {
		t.Errorf("error 2!")
	}

	s3 := []byte{0x11, 0x22, 0x33}
	const h3 uint64 = 0x1000000000000
	_, err = prependBlockHeight(h3, s3)
	if nil == err {
		t.Errorf("error 3!")
	}

	s4 := []byte("hi")
	const h4 uint64 = 0x1000000000000 - 1
	desired4 := []byte{'F', 'a', 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 'h', 'i'}
	r4, err := prependBlockHeight(h4, s4)
	if nil != err {
		t.Errorf("error 4!")
	}

	if !reflect.DeepEqual(r4, desired4) {
		t.Errorf("deep equal 4!")
		t.Logf("%x\n", desired4)
		t.Logf("%x\n", r4)
	}

	s5 := []byte{0x11, 0x22, 0x33}
	const h5 uint64 = 3
	desired5 := []byte{'F', 'a', 0, 0, 0, 0, 0, 3, 0x11, 0x22, 0x33}
	r5, err := prependBlockHeight(h5, s5)
	if nil != err {
		t.Errorf("error 5!")
	}

	if !reflect.DeepEqual(r5, desired5) {
		t.Errorf("deep equal 5!")
		t.Logf("%x\n", desired5)
		t.Logf("%x\n", r5)
	}
}

func TestWallet(t *testing.T) {
	accountMap, err := wclient.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("account & balance: ", spew.Sdump(accountMap))
	t.Log("wallet account map len = ", len(accountMap))

	allAddr := make([]btcutil.Address, 0, 1000)
	for key := range accountMap {
		addresses, err := wclient.GetAddressesByAccount(key)
		t.Log("account name=", key, ", addr=", addresses)
		if err != nil {
			t.Fatal(err)
		}
		allAddr = append(allAddr, addresses...)
	}
	t.Log("allAddr.len=", len(allAddr))

	var i int
	ca := make([]btcutil.Address, 1)
	for _, a := range allAddr {
		ca[0] = a
		balance, err := wclient.ListUnspentMinMaxAddresses(1, 999999, ca)
		if err != nil {
			t.Fatal(err)
		}
		if len(balance) > 0 {
			fmt.Print(a, "  ")
			for _, b := range balance {
				fmt.Print(b.Amount, "  ")
			}
			fmt.Println()
			i++
		}
	}
	t.Log("num of addresses with balance=", i)
}

func TestListUnspent(t *testing.T) {
	balance, err := wclient.ListUnspent()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("test.listunspent.len==%d\n", len(balance))

	var i int
	for _, b := range balance {
		if b.Amount > float64(0.0001) {
			//fmt.Println(b)
			i++
		}
	}
	t.Logf("qualified unspent len==%d\n", i)

	var included bool
	for _, b := range balance {
		if compareUnspentResult(spentResult, b) {
			included = true
			break
		}
	}

	if !included {
		t.Log("qualified unspent does NOT include the bug")
	}
}

func TestUnconfirmedSpent(t *testing.T) {
	b1, _ := wclient.ListUnspent()     //minConf=1
	b2, _ := wclient.ListUnspentMin(0) //minConf=0
	for _, b := range b1 {
		var i = len(b2) + 1
		for j, a := range b2 {
			if compareUnspentResult(a, b) {
				i = j
				break
			}
		}
		if i < len(b2)+1 {
			b2 = append(b2[:i], b2[(i+1):]...)
		}
	}
	t.Log("Unconfirmed unspent len=", len(b2))

	var sum float64
	for i := 0; i < len(b2); i++ {
		sum += b2[i].Amount
	}
	t.Log("Unconfirmed unspent sum = ", sum)
}

func TestRepeatedSpending(t *testing.T) {
	for i := 0; i < 10; i++ {
		hash, err := writeToBTC([]byte{byte(i)}, uint64(10))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("repeating=%d, hash=%s\n", i, hash)
		time.Sleep(30 * time.Second)
	}
}

func writeToBTC(bytes []byte, blockHeight uint64) (*wire.ShaHash, error) {
	for attempts := 0; attempts < maxTrials; attempts++ {
		hash := common.NewHash()
		hash.SetBytes(bytes)
		txHash, err := doTransaction(hash, blockHeight, nil) //SendRawTransactionToBTC(hash, blockHeight)
		if err != nil {
			log.Printf("Attempt %d to send raw tx to BTC failed: %s\n", attempts, err)
			time.Sleep(time.Duration(attempts*20) * time.Second)
			continue
		}
		return txHash, nil
	}
	return nil, fmt.Errorf("Fail to write hash %s to BTC. ", bytes)
}

// Initialize the level db and share it with other components
func initDB() database.Db {

	var ldbpath = "/home/bw/.factom/ldb"
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	if err != nil {
		fmt.Printf("err opening db: %v\n", err)

	}

	if db == nil {
		fmt.Println("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)

		if err != nil {
			panic(err)
		}
	}
	fmt.Println("Database started from: " + ldbpath)
	return db
}
*/
