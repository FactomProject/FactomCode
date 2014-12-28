package wallet

import (
	"os"
)

type KeyManager struct {
	keyPair *PrivateKey
	storePath string
	storeFile string
}


func (km *KeyManager) NewKeyManager(path string, file string ) (err error) {
	km.storePath = path
	km.storeFile = file

	err = km.LoadOrGenerateKeys()

	return 
}

func (km *KeyManager) LoadOrGenerateKeys() (err error) {
	file, err := os.Open(km.FilePath())
	if ( err == nil ) {
		//try loading keys from file
		if ( km.LoadKeys(file) ) { return }

		//failed to load keys, yet file exists, reopen in append mode
		file.Close()
		file, err = os.OpenFile(km.FilePath(),os.O_APPEND,os.FileMode(0666))
		if ( err != nil) { return }

	} else {

		if ( !os.IsNotExist(err) ) { return }   //panic?

		//file does not exist, check directory
		_, err = os.Stat(km.storePath)
		if ( err != nil ) {
			if (!os.IsNotExist(err)) {return} //other error. panic?

			//directory doesnt exist, create it
			err = os.MkdirAll(km.storePath, 0755)
			if ( err != nil ) {return}
		}

		//now create new file 
		file, err = os.Create(km.FilePath())
		if ( err != nil ) { return }
	}

	err = km.GenerateNewKey()
	if ( err != nil ) {return}

	//file is now open for writing, and new keys generated

	err = km.WriteKeys(file) // write keys
	file.Close()
	return  //err

}

func (km *KeyManager) LoadKeys(file *os.File) bool {
	//load file
	return false;
}

func (km *KeyManager) WriteKeys(file *os.File) (err error) {
	//write file
	return nil;
}

func (km *KeyManager) GenerateNewKey() (err error) {
	km.keyPair = new(PrivateKey)
	err = km.keyPair.GenerateKey()
	return
}

func (km *KeyManager) FilePath() (fp string) {
	return km.storePath + "/" + km.storeFile
}
