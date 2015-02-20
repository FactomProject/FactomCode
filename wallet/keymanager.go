package wallet

import (
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"os"
)

type KeyManager struct {
	keyPair   *notaryapi.PrivateKey
	storePath string
	storeFile string
}

func (km *KeyManager) InitKeyManager(path string, file string) (err error) {
	km.storePath = path
	km.storeFile = file

	err = km.LoadOrGenerateKeys()

	return
}

func (km *KeyManager) LoadOrGenerateKeys() (err error) {
	file, err := os.Open(km.FilePath())

	defer func() {
		file.Close()
	}()

	if err == nil {
		//try loading keys from file
		err = km.LoadKeys(file)
		if err == nil {
			return
		} //we are good **

		//failed to load keys, yet file exists, reopen in append mode
		file.Close()
		file, err = os.OpenFile(km.FilePath(), os.O_APPEND, os.FileMode(0666))
		if err != nil {
			return
		}

	} else { // err != nil - probably file not exist

		if !os.IsNotExist(err) {
			return
		} //panic?

		//file does not exist, check directory
		_, err = os.Stat(km.storePath)
		if err != nil {
			if !os.IsNotExist(err) {
				return
			} //other error. panic?

			//directory doesnt exist, create it
			err = os.MkdirAll(km.storePath, 0755)
			if err != nil {
				return
			}
		}

		//now create new file
		file, err = os.Create(km.FilePath())
		if err != nil {
			return
		}
	}

	err = km.GenerateNewKey()
	if err == nil {
		//file is now open for writing, and new keys generated
		err = km.WriteKeys(file) // write keys
	}

	return //err
}

func (km *KeyManager) LoadKeys(file *os.File) (err error) {
	//load file
	km.keyPair = new(notaryapi.PrivateKey)
	km.keyPair.AllocateNew()

	n, err := file.Read(km.keyPair.Key[:])
	if err == nil {
		if n != 64 {
			err = errors.New(" n != ed25519.PrivateKeySize ")
		}
	}

	if err != nil {
		return
	}

	n, err = file.Read(km.keyPair.Pub.Key[:])
	if err == nil {
		if n != 32 {
			err = errors.New(" n != ed25519.PublicSize ")
		}
	}

	return
}

func (km *KeyManager) WriteKeys(file *os.File) (err error) {
	//write file
	n, err := file.Write(km.keyPair.Key[:])
	if err == nil {
		if n != 64 {
			err = errors.New(" n != ed25519.PrivateKeySize ")
		}
	}
	n, err = file.Write(km.keyPair.Pub.Key[:])
	if err == nil {
		if n != 32 {
			err = errors.New(" n != ed25519.PublicSize ")
		}
	}

	return
}

func (km *KeyManager) GenerateNewKey() (err error) {
	km.keyPair = new(notaryapi.PrivateKey)
	err = km.keyPair.GenerateKey()
	return
}

func (km *KeyManager) FilePath() (fp string) {
	return km.storePath + "/" + km.storeFile
}
