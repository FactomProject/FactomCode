package wallet

import (
	"log"
	"os"

	"code.google.com/p/gcfg"
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

func SignData(data []byte) Signature {
	return keyManager.keyPair.Sign(data)
}

//impliment Signer
func Sign(d []byte) Signature { return SignData(d) }

func ClientPublicKey() PublicKey {
	return keyManager.keyPair.Pub
}
