// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package factomapi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wallet"
	"github.com/FactomProject/btcd"
	"github.com/FactomProject/btcd/wire"
)

//to be improved
var (
	serverAddr      = "localhost:8083"
	db              database.Db
	creditsPerChain uint32              = 10
	inMsgQueue     chan<- wire.FtmInternalMsg //outgoing message queue for factom application messages
)

// This method will be replaced with a Factoid transaction once we have the factoid implementation in place
func BuyEntryCredit(version uint16, ecPubKey *common.Hash, from *common.Hash, value uint64, fee uint64, sig *common.Signature) error {

	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("!!! NOT IMPLEMENTED !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	// XXX TODO FIXME: gotta be linked to Factoid -- there is no buy or get credit P2P message
	/*
			msgGetCredit := wire.NewMsgGetCredit()
			msgGetCredit.ECPubKey = ecPubKey
				msgGetCredit.FactoidBase = value

		outMsgQueue <- msgGetCredit
	*/

	return nil
}

//func FactoidTx(addr factoid.Address, n uint32) {
//	var msg wire.MsgTx
//	genb := factoid.FactoidGenesis(wire.TestNet)
//	outs := factoid.OutputsTx(&genb.Transactions[0])
//	txm := factoid.NewTxFromOutputToAddr(genb.Transactions[0].Id(), outs,
//		n, factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
//	ds := wallet.DetachMarshalSign(txm.TxData)
//	ss := factoid.NewSingleSignature(ds)
//	factoid.AddSingleSigToTxMsg(txm, ss)
//
//	msg.Data, err := txm.MarshalBinary()
//	if err != nil {
//		fmt.Println(err)
//	}
//	outMsgQueue <- msg
//}

func GetEntryCreditBalance(ecPubKey *common.Hash) (credits int32, err error) {

	return  btcd.GetEntryCreditBalance(ecPubKey)
}

func GetChainByHashStr(id string) (*common.EChain, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchChainByHash(hash)
}

func GetAllChains() ([]common.EChain, error) {
	return db.FetchAllChains()
}

func GetDirectoryBloks(fromBlockHeight uint64, toBlockHeight uint64) (dBlocks []common.DBlock, err error) {
	//needs to be improved ??
	dBlocks, _ = db.FetchAllDBlocks()
	sort.Sort(byBlockID(dBlocks))

	if fromBlockHeight > uint64(len(dBlocks)-1) {
		return nil, nil
	} else if toBlockHeight > uint64(len(dBlocks)-1) {
		toBlockHeight = uint64(len(dBlocks) - 1)
	}

	return dBlocks[fromBlockHeight : toBlockHeight+1], nil
}

func GetDirectoryBlokByHashStr(addr string) (*common.DBlock, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchDBlockByHash(hash)
}

func GetDBInfoByHashStr(addr string) (*common.DBInfo, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchDBInfoByHash(hash)
}

func GetEntryBlokByHashStr(addr string) (*common.EBlock, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEBlockByHash(hash)
}

func GetEntryBlokByMRStr(addr string) (*common.EBlock, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEBlockByMR(hash)
}

func GetBlokHeight() (int, error) {
	b := make([]common.DBlock, 0)
	b, err := db.FetchAllDBlocks()
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func GetEntriesByExtID(eid string) (entries []common.Entry, err error) {
	extIDMap, err := db.InitializeExternalIDMap()
	if err != nil {
		return nil, err
	}
	for key, _ := range extIDMap {
		if strings.Contains(key[32:], eid) {
			hash := new(common.Hash)
			hash.Bytes = []byte(key[:32])
			entry, err := db.FetchEntryByHash(hash)
			if err != nil {
				return entries, err
			}
			entries = append(entries, *entry)
		}
	}
	return entries, err
}

func GetEntryByHashStr(addr string) (*common.Entry, error) {
	hash := new(common.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEntryByHash(hash)
}

//func GetDirectoryBlokByHash(dBlockHash *common.Hash) (dBlock *common.DBlock, err error) {
//
//	dBlock, err = db.FetchDBlockByHash(dBlockHash)
//
//	return dBlock, err
//}

//func GetEntryBlokByHash(eBlockHash *common.Hash) (eBlock *common.EBlock, err error) {
//
//	eBlock, err = db.FetchEBlockByHash(eBlockHash)
//
//	return eBlock, err
//}

//func GetEntryByHash(entrySha *common.Hash) (entry *common.Entry, err error) {
//
//	entry, err = db.FetchEntryByHash(entrySha)
//
//	return entry, err
//}

// to be removed------------------------------
func SetServerAddr(addr string) error {
	serverAddr = addr

	return nil
}

func SetDB(database database.Db) error {
	db = database

	return nil
}
func SetInMsgQueue(inMsgQ chan<- wire.FtmInternalMsg) error {
	inMsgQueue = inMsgQ

	return nil
}

//-=-----------------------------------------

// array sorting implementation
type byBlockID []common.DBlock

func (f byBlockID) Len() int {
	return len(f)
}
func (f byBlockID) Less(i, j int) bool {
	return f[i].Header.BlockID < f[j].Header.BlockID
}
func (f byBlockID) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// array sorting implementation
type byEBlockID []common.EBlock

func (f byEBlockID) Len() int {
	return len(f)
}
func (f byEBlockID) Less(i, j int) bool {
	return f[i].Header.EBHeight > f[j].Header.EBHeight
}
func (f byEBlockID) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//-------------------------------------
/*

// PrintEntry is a helper function for debugging entry transport and encoding
func PrintEntry(e *Entry) {
	fmt.Println("ChainID:", hex.EncodeToString(e.ChainID))
	fmt.Println("ExtIDs:")
	for i := range e.ExtIDs {
		fmt.Println("	", string(e.ExtIDs[i]))
	}
	fmt.Println("Data:", string(e.Data))
}

// NewEntry creates a factom entry. It is supplied a string chain id, a []byte
// of data, and a series of string external ids for entry lookup
func NewEntry(cid string, eids []string, data []byte) (e *Entry, err error) {
	e = new(Entry)
	e.ChainID, err = hex.DecodeString(cid)
	if err != nil {
		return nil, err
	}
	e.Data = data
	for _, v := range eids {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	return
}

// NewChain creates a factom chain from a []string chain name and a new entry
// to be the first entry of the new chain from []byte data, and a series of
// string external ids
func NewChain(name []string, eids []string, data []byte) (c *Chain, err error) {
	c = new(Chain)
	for _, v := range name {
		c.Name = append(c.Name, []byte(v))
	}
	c.GenerateID()
	e := new(Entry)
	e.ChainID = c.ChainID
	e.Data = data
	for _, v := range eids {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	c.FirstEntry = e
	return
}

// CommitEntry sends a message to the factom network containing a hash of the
// entry to be used to verify the later RevealEntry.
func CommitEntry(e *Entry) error {
	data := url.Values{
		"datatype": {"entryhash"},
		"format":   {"binary"},
		"data":     {e.Hash()},
	}
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}

// RevealEntry sends a message to the factom network containing the binary
// encoded entry for the server to add it to the factom blockchain. The entry
// will be rejected if a CommitEntry was not done.
func RevealEntry(e *Entry) error {
	data := url.Values{
		"datatype": {"entry"},
		"format":   {"binary"},
		"entry":    {e.Hex()},
	}
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}
*/
// CommitChain sends a message to the factom network containing a series of
// hashes to be used to verify the later RevealChain.
func CommitChain(c *common.EChain) error {
	util.Trace()
	var buf bytes.Buffer

	// Calculate the required credits
	bChain, _ := c.MarshalBinary()
	credits := uint32(binary.Size(bChain)/1000+1) + creditsPerChain

	binaryEntry, _ := c.FirstEntry.MarshalBinary()
	entryHash := common.Sha(binaryEntry)

	entryChainIDHash := common.Sha(append(c.ChainID.Bytes, entryHash.Bytes...))

	// Create a msg signature (timestamp + chainid + entry hash + entryChainIDHash + credits)
	timestamp := uint64(time.Now().Unix())
	binary.Write(&buf, binary.BigEndian, timestamp)
	buf.Write(c.ChainID.Bytes)
	buf.Write(entryHash.Bytes)
	buf.Write(entryChainIDHash.Bytes)
	binary.Write(&buf, binary.BigEndian, credits)
	sig := wallet.SignData(buf.Bytes())

	//Construct a msg and add it to the msg queue
	msgCommitChain := wire.NewMsgCommitChain()
	msgCommitChain.ChainID = c.ChainID
	msgCommitChain.Credits = credits
	msgCommitChain.ECPubKey = new(common.Hash)
	msgCommitChain.ECPubKey.Bytes = (*sig.Pub.Key)[:]
	msgCommitChain.EntryChainIDHash = entryChainIDHash
	msgCommitChain.EntryHash = entryHash
	msgCommitChain.Sig = (*sig.Sig)[:]
	msgCommitChain.Timestamp = timestamp

	inMsgQueue <- msgCommitChain

	return nil
}

// RevealChain sends a message to the factom network containing the binary
// encoded first entry for a chain to be used by the server to add a new factom
// chain. It will be rejected if a CommitChain was not done.
func RevealChain(c *common.EChain) error {

	//Construct a msg and add it to the msg queue
	msgRevealChain := wire.NewMsgRevealChain()
	msgRevealChain.Chain = c

	inMsgQueue <- msgRevealChain

	return nil
}

// PrintEntry is a helper function for debugging entry transport and encoding
func PrintEntry(e *Entry) {
	fmt.Println("ChainID:", hex.EncodeToString(e.ChainID))
	fmt.Println("ExtIDs:")
	for _, v := range e.ExtIDs {
		fmt.Println("	", string(v))
	}
	fmt.Println("Data:", string(e.Data))
}

// NewEntry creates a factom entry. It is supplied a string chain id, a []byte
// of data, and a series of string external ids for entry lookup
func NewEntry(cid string, eids []string, data []byte) (e *Entry, err error) {
	e = new(Entry)
	e.ChainID, err = hex.DecodeString(cid)
	if err != nil {
		return nil, err
	}
	e.Data = data
	for _, v := range eids {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	return
}

// NewChain creates a factom chain from a []string chain name and a new entry
// to be the first entry of the new chain from []byte data, and a series of
// string external ids
func NewChain(name []string, eids []string, data []byte) (c *Chain, err error) {
	c = new(Chain)
	for _, v := range name {
		c.Name = append(c.Name, []byte(v))
	}
	str_name := c.GenerateID()
	c.FirstEntry, err = NewEntry(str_name, eids, data)
	return
}

// CommitEntry sends a message to the factom network containing a hash of the
// entry to be used to verify the later RevealEntry.
func CommitEntry(e *common.Entry) error {
	util.Trace()
	var buf bytes.Buffer

	bEntry, _ := e.MarshalBinary()
	entryHash := common.Sha(bEntry)
	// Calculate the required credits
	credits := uint32(binary.Size(bEntry)/1000 + 1)

	// Create a msg signature (timestamp + entry hash + credits)
	timestamp := uint64(time.Now().Unix())
	binary.Write(&buf, binary.BigEndian, timestamp)
	buf.Write(entryHash.Bytes)
	binary.Write(&buf, binary.BigEndian, credits)
	sig := wallet.SignData(buf.Bytes())

	//Construct a msg and add it to the msg queue
	msgCommitEntry := wire.NewMsgCommitEntry()
	msgCommitEntry.Credits = credits
	msgCommitEntry.ECPubKey = new(common.Hash)
	msgCommitEntry.ECPubKey.Bytes = (*sig.Pub.Key)[:]
	msgCommitEntry.EntryHash = entryHash
	msgCommitEntry.Sig = (*sig.Sig)[:]
	msgCommitEntry.Timestamp = timestamp

	util.Trace()
	inMsgQueue <- msgCommitEntry
	util.Trace()

	return nil
}

// RevealEntry sends a message to the factom network containing the binary
// encoded entry for the server to add it to the factom blockchain. The entry
// will be rejected if a CommitEntry was not done.
func RevealEntry(e *common.Entry) error {

	//Construct a msg and add it to the msg queue
	msgRevealEntry := wire.NewMsgRevealEntry()
	msgRevealEntry.Entry = e

	inMsgQueue <- msgRevealEntry

	return nil
}

func SubmitFactoidTx(m wire.Message) error {

	inMsgQueue <- m

	return nil
}
