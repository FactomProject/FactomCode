package wallet

import (
	"testing"
)

func TestKeyManager(t *testing.T) {

	walletfile := "wallet.dat"
	walletstorepath := "c:/tmpxxxx/wallet"

	//defaultPrivKey PrivateKey
	keymanager := new(KeyManager)

	err := keymanager.InitKeyManager(walletstorepath, walletfile)
	if err != nil {
		t.Fatalf("err")
	}

	priv := keymanager.keyPair
	msg := "Test Message Sign"
	sig := priv.Sign([]byte(msg))
	if sig.Sig == nil {
		t.Fatalf("bad Sig")
	}
	t.Logf("TestKeyManager Sig: %v", sig.Sig)

	if sig.Pub.Key == nil {
		t.Fatalf("bad Pub.Key")
	}
	t.Logf("TestKeyManager Pub.Key: %v", sig.Pub.Key)

	if !sig.Verify([]byte(msg)) {
		t.Fatalf("TestKeyManager sig.Verify retuned false")
	}

	//walletFile = "wallet2.dat"

	keymanager = new(KeyManager)

	err = keymanager.InitKeyManager(walletstorepath, walletfile)
	if err != nil {
		t.Fatalf("err")
	}

	priv2 := keymanager.keyPair
	sig2 := priv2.Sign([]byte(msg))
	if sig2.Sig == nil {
		t.Fatalf("bad Sig")
	}
	t.Logf("TestKeyManager Sig2: %v", sig2.Sig)

	if sig2.Pub.Key == nil {
		t.Fatalf("bad Pub.Key")
	}
	t.Logf("TestKeyManager Pub.Key2: %v", sig2.Pub.Key)

	if !sig2.Verify([]byte(msg)) {
		t.Fatalf("TestKeyManager sig2.Verify retuned false")
	}

	sig.Pub = sig2.Pub
	if !sig.Verify([]byte(msg)) {
		t.Fatalf("3 TestKeyManager sig.Verify retuned false")
	}
}

func TestSignData(t *testing.T) {
	msg := "Test Message Sign"

	if sig := SignData([]byte(msg)); !sig.Verify([]byte(msg)) {
		t.Fatalf("TestSignData SignData sig.Verify retuned false")
	}

	if sig := Sign([]byte(msg)); !sig.Verify([]byte(msg)) {
		t.Fatalf("TestSignData Sign sig.Verify retuned false")
	}
}

func TestGetAddress(t *testing.T) {
	t.Logf("Address: %v", FactoidAddress())

	t.Logf("ClientPublicKeyStr: %v", ClientPublicKeyStr())

}
