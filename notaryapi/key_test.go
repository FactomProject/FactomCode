package notaryapi

import (
	"testing"
)

func TestGenerateKey(t *testing.T) {
	priv := new(PrivateKey)

	err := priv.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}

	if priv.Key == nil {
		t.Fatalf("bad Key")
	}
	t.Logf("PrivateKey: %v", priv.Key)

	if priv.Pub.Key == nil {
		t.Fatalf("bad Pub.Key")
	}
	t.Logf("Pub.Key: %v", priv.Pub.Key)
}

func TestSign(t *testing.T) {
	priv := new(PrivateKey)

	err := priv.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}

	msg := "Test Message Sign"

	sig := priv.Sign([]byte(msg))
	if sig.Sig == nil {
		t.Fatalf("bad Sig")
	}
	t.Logf("Sig: %v", sig.Sig)

	if sig.Pub.Key == nil {
		t.Fatalf("bad Pub.Key")
	}
	t.Logf("Pub.Key: %v", sig.Pub.Key)

	if !sig.Verify([]byte(msg)) {
		t.Fatalf("sig.Verify retuned false")
	}
}

func TestVerify(t *testing.T) {
	priv1 := new(PrivateKey)
	priv2 := new(PrivateKey)

	err := priv1.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = priv2.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}

	msg1 := "Test Message Sign1"
	msg2 := "Test Message Sign2"

	sig11 := priv1.Sign([]byte(msg1))
	sig12 := priv1.Sign([]byte(msg2))
	sig21 := priv2.Sign([]byte(msg1))
	sig22 := priv2.Sign([]byte(msg2))

	if !sig11.Verify([]byte(msg1)) {
		t.Fatalf("sig11.Verify retuned false")
	}

	if sig11.Verify([]byte(msg2)) {
		t.Fatalf("sig11.Verify retuned true")
	}

	if !sig12.Verify([]byte(msg2)) {
		t.Fatalf("sig12.Verify retuned false")
	}

	if sig12.Verify([]byte(msg1)) {
		t.Fatalf("sig12.Verify retuned true")
	}

	if !sig21.Verify([]byte(msg1)) {
		t.Fatalf("sig21.Verify retuned false")
	}

	//same pub key
	sig21.Pub = sig22.Pub
	if !sig21.Verify([]byte(msg1)) {
		t.Fatalf("sig21.Verify retuned false")
	}

	//wrong pub key
	sig21.Pub = priv1.Pub
	if sig21.Verify([]byte(msg1)) {
		t.Fatalf("sig21.Verify retuned true")
	}

	if !sig22.Verify([]byte(msg2)) {
		t.Fatalf("sig22.Verify retuned false")
	}

	//wrong sig
	sig22.Sig = sig12.Sig
	if sig22.Verify([]byte(msg2)) {
		t.Fatalf("sig22.Verify retuned true")
	}

	if !priv1.Pub.Verify([]byte(msg1), sig11.Sig) {
		t.Fatalf("Pub.Verify retuned false")
	}

	if !Verify(priv1.Pub.Key, []byte(msg1), sig11.Sig) {
		t.Fatalf("Verify retuned false")
	}

	if !VerifySlice(priv1.Pub.Key[:], []byte(msg1), sig11.Sig[:]) {
		t.Fatalf("VerifySlice retuned false")
	}

}
