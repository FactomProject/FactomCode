package process

import (
	"fmt"
	"reflect"
	"testing"

	/*
		"time"
		"bytes"

		"github.com/FactomProject/btcutil"
		"github.com/FactomProject/btcd/wire"
		"github.com/FactomProject/FactomCode/common"
	*/)

func TestPrependBlockHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	fmt.Println("testing...")

	s1 := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	const h1 uint64 = 0x123456789ABC // temp block height

	_, err := prependBlockHeight(h1, s1)
	if nil != err {
		t.Errorf("error 1!")
	}

	const h2 uint64 = 0
	_, err = prependBlockHeight(h2, s1)
	if nil == err {
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
		fmt.Printf("%x\n", desired4)
		fmt.Printf("%x\n", r4)
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
		fmt.Printf("%x\n", desired5)
		fmt.Printf("%x\n", r5)
	}
}

/*

func init() {
	err := initRPCClient()
	if err != nil {
		fmt.Println(err.Error())
	}

	if err := initWallet(); err != nil {
		fmt.Println(err.Error())
	}
}

func TestWallet(t *testing.T) {
	accountMap, err := client.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
//	fmt.Println("account & balance: ", accountMap)
	if accountMap[""] <= 0 {
		t.Errorf("wallet account name is not empty")
	}
	if len(accountMap) != 1 {
		t.Errorf("wallet account map len =%d", len(accountMap), ", is not empty")
	}

	allAddr := make([]btcutil.Address, 0, 1000)
	for key, _ := range accountMap {
		addresses, err := client.GetAddressesByAccount(key)
		if err != nil {
			t.Fatal(err)
		}
		allAddr = append(allAddr, addresses...)
	}
	if len(allAddr) != 268 {
		t.Errorf("allAddr.len=%d", len(allAddr), ", NOT 268")
	}

	var i int
	ca := make([]btcutil.Address, 1)
	for _, a := range allAddr {
		ca[0] = a
		balance, err := client.ListUnspentMinMaxAddresses(1, 999999, ca)
		if err != nil {
			t.Fatal(err)
		}
		if len(balance) > 0 {
//			fmt.Print(a, "  ")
//			for _, b := range balance {
//				fmt.Print(b.Amount, "  ")
//			}
//			fmt.Println()
			i++
		}
	}
	if i != 87 {
		t.Errorf("num of addresses with balance=%d", i, ", NOT 87")
	}
}

func TestListUnspent(t *testing.T) {
	balance, err := client.ListUnspent()
	if err != nil {
		t.Fatal(err)
	}
	if len(balance) != 93 {
		t.Errorf("test.listunspent.len==%d", len(balance), ", NOT 93")
	}

	var i int
	for _, b := range balance {
		if b.Amount > float64(0.1) {
			//fmt.Println(b)
			i++
		}
	}
	if i != 21 {
		t.Errorf("qualified unspent len==%d", i, ", NOT 21")
	}

	var included bool
	for _, b := range balance {
		if compareUnspentResult(spentResult, b) {
			included = true
			break
		}
	}
	if !included {
		t.Errorf("qualified unspent does NOT include the bug")
	}
}


func TestUnconfirmedSpent(t *testing.T) {
	b1, _ := client.ListUnspent()	//minConf=1
	b2, _ := client.ListUnspentMin(0)	//minConf=0

	for _, b := range b1 {
		var i int = len(b2) + 1
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

	if len(b2) != 4 {
		t.Errorf("Unconfirmed unspent len=%d", len(b2), ", NOT 4")
	}
	var sum float64
	for i:=0; i<len(b2); i++ {
		sum += b2[i].Amount
	}
	// the same as unconfirmed balance in OnAccountBalance call back
	if sum != 1.9936741999999998 {
		t.Errorf("Unconfirmed unspent sum = %f", sum, ", not 1.9936742")
	}

}


func TestRepeatedSpending(t *testing.T) {
	for i:=0; i<100; i++ {
		hash, err := writeToBTC(common.Sha([]byte{byte(i)}))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("repeating=", i, ", hash= ", hash, "\n")
		time.Sleep(30 * time.Second)
	}
}


func TestToHash(t *testing.T) {
	s := "e517043a9770aacc7406db5f2ae8b3d687ce9bca3c8f76bc0be1ed18aed7ad68"
	txHash, err := wire.NewShaHashFromStr(s)
	if err != nil {fmt.Println(err.Error()) }
	h := toHash(txHash)
	fmt.Println("txHash=", txHash.String(), ", toHash=", h.String())
	fmt.Println("equal in string: ", txHash.String() == h.String())
	fmt.Println("equal in bytes: ", bytes.Compare(txHash.Bytes(), h.Bytes))
}

*/
