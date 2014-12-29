package wallet

import (
	"log"
	"os"
	//"fmt"
	"code.google.com/p/gcfg"
)

var (
	walletFile = "wallet.dat"
	walletStorePath = "/tmp/wallet"

	//defaultPrivKey PrivateKey
	keyManager KeyManager
)


func init() {
	loadConfigurations()
	loadKeys()
}

func loadKeys() {
	err := keyManager.InitKeyManager(walletStorePath,walletFile)
	if ( err != nil) {
		panic(err)
	}

}


func loadConfigurations(){
	cfg := struct {
		Wallet struct{
			WalletStorePath	string		
	    }
    }{}

	var  sf = "wallet.conf"	
	wd, err := os.Getwd()
	if err != nil{
		log.Println(err)
	} else {
		sf =  wd+"/"+sf		
	}	

	err = gcfg.ReadFileInto(&cfg, sf)
	if err != nil{
		log.Println(err)
		log.Println("Wallet using default settings...")
	} else {
		log.Println("Walet using settings from: " + sf)
		log.Println(cfg)
	
		walletStorePath = cfg.Wallet.WalletStorePath
	}
	
}

/*
func SignData1(data []byte) (signed []byte, pubkey []byte) {
	sig := keyManager.keyPair.Sign(data)

	signed = (*sig.Sig)[:]
	pubkey = (*sig.Pub.Key)[:]
	return 
}
*/

func SignData(data []byte) Signature {
	return keyManager.keyPair.Sign(data)
}

