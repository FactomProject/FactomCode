package wallet

import (
	"log"
	"os"
	//"fmt"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/notaryapi"
)

var (
	walletFile      = "wallet.dat"
	walletStorePath = "/tmp/wallet"

	//defaultPrivKey PrivateKey
	keyManager KeyManager
)

func init() {
	loadConfigurations()
	loadKeys()
}

func loadKeys() {
	err := keyManager.InitKeyManager(walletStorePath, walletFile)
	if err != nil {
		panic(err)
	}

}

func loadConfigurations() {
	cfg := struct {
		Wallet struct {
			WalletStorePath string
		}
	}{}

	var sf = "wallet.conf"
	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	} else {
		sf = wd + "/" + sf
	}

	err = gcfg.ReadFileInto(&cfg, sf)
	if err != nil {
		log.Println(err)
		log.Println("Wallet using default settings...")
	} else {
		log.Println("Walet using settings from: " + sf)
		log.Println(cfg)

		walletStorePath = cfg.Wallet.WalletStorePath
	}

}

func SignData(data []byte) notaryapi.Signature {
	return keyManager.keyPair.Sign(data)
}

//impliment Signer
func Sign(d []byte) notaryapi.Signature { return SignData(d) }

func ClientPublicKey() notaryapi.PublicKey {
	return keyManager.keyPair.Pub
}

func MarshalSign(msg notaryapi.BinaryMarshallable) notaryapi.Signature {
	return keyManager.keyPair.MarshalSign(msg)
}

func DetachMarshalSign(msg notaryapi.BinaryMarshallable) *notaryapi.DetachedSignature {
	sig := MarshalSign(msg)
	return sig.DetachSig()
}

func FactoidAddress() string {
	netid := byte('\x07')
	return factoid.AddressFromPubKey(ClientPublicKey().Key, netid)
}

func ClientPublicKeyStr() string {
	return ClientPublicKey().String()
}
