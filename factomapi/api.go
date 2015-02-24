package factomapi

import (
	//"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/FactomProject/FactomCode/database"
	//	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/wallet"
	"net/http"
	"net/url"
	"sort"
	//"github.com/FactomProject/FactomCode/database/ldb"
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"time"
)

//to be improved
var (
	serverAddr      = "localhost:8083"
	db              database.Db
	creditsPerChain uint32                    = 10
	outMsgQueue     chan<- factomwire.Message //outgoing message queue for factom application messages
)

// This method will be replaced with a Factoid transaction once we have the factoid implementation in place
func BuyEntryCredit(version uint16, ecPubKey *notaryapi.Hash, from *notaryapi.Hash, value uint64, fee uint64, sig *notaryapi.Signature) error {

	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("!!! NOT IMPLEMENTED !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	// XXX TODO FIXME: gotta be linked to Factoid -- there is no buy or get credit P2P message
	/*
		msgGetCredit := factomwire.NewMsgGetCredit()
		msgGetCredit.ECPubKey = ecPubKey
		msgGetCredit.FactoidBase = value

		outMsgQueue <- msgGetCredit
	*/

	return nil
}

//func FactoidTx(addr factoid.Address, n uint32) {
//	var msg factomwire.MsgTx
//	genb := factoid.FactoidGenesis(factomwire.TestNet)
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

func GetEntryCreditBalance(ecPubKey *notaryapi.Hash) (credits int32, err error) {
	data := url.Values{}
	data.Set("format", "binary")
	data.Set("datatype", "getbalance")
	data.Set("ECPubKey", ecPubKey.String())

	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	resp, err := http.PostForm(server, data)

	contents, err := ioutil.ReadAll(resp.Body)

	buf := bytes.NewBuffer(contents)
	binary.Read(buf, binary.BigEndian, &credits)

	return credits, err
}

func GetDirectoryBloks(fromBlockHeight uint64, toBlockHeight uint64) (dBlocks []notaryapi.DBlock, err error) {
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

func GetDirectoryBlokByHashStr(addr string) (*notaryapi.DBlock, error) {
	hash := new(notaryapi.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchDBlockByHash(hash)
}

func GetEntryBlokByHashStr(addr string) (*notaryapi.EBlock, error) {
	hash := new(notaryapi.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEBlockByHash(hash)
}

func GetEntryBlokByMRStr(addr string) (*notaryapi.EBlock, error) {
	hash := new(notaryapi.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEBlockByMR(hash)
}

func GetEntryByHashStr(addr string) (*notaryapi.Entry, error) {
	hash := new(notaryapi.Hash)
	a, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	hash.Bytes = a

	return db.FetchEntryByHash(hash)
}

//func GetDirectoryBlokByHash(dBlockHash *notaryapi.Hash) (dBlock *notaryapi.DBlock, err error) {
//
//	dBlock, err = db.FetchDBlockByHash(dBlockHash)
//
//	return dBlock, err
//}

//func GetEntryBlokByHash(eBlockHash *notaryapi.Hash) (eBlock *notaryapi.EBlock, err error) {
//
//	eBlock, err = db.FetchEBlockByHash(eBlockHash)
//
//	return eBlock, err
//}

//func GetEntryByHash(entrySha *notaryapi.Hash) (entry *notaryapi.Entry, err error) {
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
func SetOutMsgQueue(outMsgQ chan<- factomwire.Message) error {
	outMsgQueue = outMsgQ

	return nil
}

//-=-----------------------------------------

// array sorting implementation
type byBlockID []notaryapi.DBlock

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
type byEBlockID []notaryapi.EBlock

func (f byEBlockID) Len() int {
	return len(f)
}
func (f byEBlockID) Less(i, j int) bool {
	return f[i].Header.BlockID > f[j].Header.BlockID
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
func CommitChain(c *notaryapi.EChain) error {
	var buf bytes.Buffer

	// Calculate the required credits
	bChain, _ := c.MarshalBinary()
	credits := uint32(binary.Size(bChain)/1000+1) + creditsPerChain

	binaryEntry, _ := c.FirstEntry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)

	entryChainIDHash := notaryapi.Sha(append(c.ChainID.Bytes, entryHash.Bytes...))

	// Create a msg signature (timestamp + chainid + entry hash + entryChainIDHash + credits)
	timestamp := uint64(time.Now().Unix())
	binary.Write(&buf, binary.BigEndian, timestamp)
	buf.Write(c.ChainID.Bytes)
	buf.Write(entryHash.Bytes)
	buf.Write(entryChainIDHash.Bytes)
	binary.Write(&buf, binary.BigEndian, credits)
	sig := wallet.SignData(buf.Bytes())

	//Construct a msg and add it to the msg queue
	msgCommitChain := factomwire.NewMsgCommitChain()
	msgCommitChain.ChainID = c.ChainID
	msgCommitChain.Credits = credits
	msgCommitChain.ECPubKey = new(notaryapi.Hash)
	msgCommitChain.ECPubKey.Bytes = (*sig.Pub.Key)[:]
	msgCommitChain.EntryChainIDHash = entryChainIDHash
	msgCommitChain.EntryHash = entryHash
	msgCommitChain.Sig = (*sig.Sig)[:]
	msgCommitChain.Timestamp = timestamp

	outMsgQueue <- msgCommitChain

	return nil
}

// RevealChain sends a message to the factom network containing the binary
// encoded first entry for a chain to be used by the server to add a new factom
// chain. It will be rejected if a CommitChain was not done.
func RevealChain(c *notaryapi.EChain) error {

	//Construct a msg and add it to the msg queue
	msgRevealChain := factomwire.NewMsgRevealChain()
	msgRevealChain.Chain = c

	outMsgQueue <- msgRevealChain

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
func CommitEntry(e *notaryapi.Entry) error {
	var buf bytes.Buffer

	bEntry, _ := e.MarshalBinary()
	entryHash := notaryapi.Sha(bEntry)
	// Calculate the required credits
	credits := uint32(binary.Size(bEntry)/1000 + 1)

	// Create a msg signature (timestamp + entry hash + credits)
	timestamp := uint64(time.Now().Unix())
	binary.Write(&buf, binary.BigEndian, timestamp)
	buf.Write(entryHash.Bytes)
	binary.Write(&buf, binary.BigEndian, credits)
	sig := wallet.SignData(buf.Bytes())

	//Construct a msg and add it to the msg queue
	msgCommitEntry := factomwire.NewMsgCommitEntry()
	msgCommitEntry.Credits = credits
	msgCommitEntry.ECPubKey = new(notaryapi.Hash)
	msgCommitEntry.ECPubKey.Bytes = (*sig.Pub.Key)[:]
	msgCommitEntry.EntryHash = entryHash
	msgCommitEntry.Sig = (*sig.Sig)[:]
	msgCommitEntry.Timestamp = timestamp

	outMsgQueue <- msgCommitEntry

	return nil
}

// RevealEntry sends a message to the factom network containing the binary
// encoded entry for the server to add it to the factom blockchain. The entry
// will be rejected if a CommitEntry was not done.
func RevealEntry(e *notaryapi.Entry) error {

	//Construct a msg and add it to the msg queue
	msgRevealEntry := factomwire.NewMsgRevealEntry()
	msgRevealEntry.Entry = e

	outMsgQueue <- msgRevealEntry

	return nil
}

// CommitEntry sends a message to the factom network containing a hash of the
// entry to be used to verify the later RevealEntry.
func CommitEntry2(e *Entry) error {
	var msg bytes.Buffer

	binary.Write(&msg, binary.BigEndian, uint64(time.Now().Unix()))
	msg.Write([]byte(e.Hash()))

	sig := wallet.SignData(msg.Bytes())
	// msg.Bytes should be a int64 timestamp followed by a binary entry

	data := url.Values{
		//		"datatype":  {"commitentry"},
		"datatype":  {factomwire.CmdCommitEntry},
		"format":    {"binary"},
		"signature": {hex.EncodeToString((*sig.Sig)[:])},
		"pubkey":    {hex.EncodeToString((*sig.Pub.Key)[:])},
		"data":      {hex.EncodeToString(msg.Bytes())},
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
func RevealEntry2(e *Entry) error {
	data := url.Values{
		"datatype": {"revealentry"},
		"format":   {"binary"},
		"entry":    {hex.EncodeToString(e.MarshalBinary())},
	}

	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}

// CommitChain sends a message to the factom network containing a series of
// hashes to be used to verify the later RevealChain.
/*func CommitChain(c *Chain) error {
	var msg bytes.Buffer

	binary.Write(&msg, binary.BigEndian, uint64(time.Now().Unix()))
	msg.Write(c.MarshalBinary())

	chainhash, chainentryhash, entryhash := c.Hash()
	msg.Write([]byte(chainhash))
	msg.Write([]byte(chainentryhash))
	msg.Write([]byte(entryhash))

	sig := wallet.SignData(msg.Bytes())

	data := url.Values{
		"datatype": {"commitchain"},
		"format":   {"binary"},
		"signature": {hex.EncodeToString((*sig.Sig)[:])},
		"pubkey": 	{hex.EncodeToString((*sig.Pub.Key)[:])},
		"data":      {hex.EncodeToString(msg.Bytes())},

	}

	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}

// RevealChain sends a message to the factom network containing the binary
// encoded first entry for a chain to be used by the server to add a new factom
// chain. It will be rejected if a CommitChain was not done.
func RevealChain(c *Chain) error {
	data := url.Values{
		"datatype": {"entry"},
		"format":   {"binary"},
		"data":     {hex.EncodeToString(c.MarshalBinary())},
	}
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}
*/
// Submit wraps CommitEntry and RevealEntry. Submit takes a FactomWriter (an
// entry is a FactomWriter) and does a commit and reveal for the entry adding
// it to the factom blockchain.
func Submit(f FactomWriter) (err error) {
	e := f.CreateFactomEntry()
	err = CommitEntry2(e)
	if err != nil {
		return err
	}
	err = RevealEntry2(e)
	if err != nil {
		return err
	}
	return nil
}

// CreateChain takes a FactomChainer (a Chain is a FactomChainer) and calls
// commit and reveal to create the factom chain on the network.
/*func CreateChain(f FactomChainer) error {
	c := f.CreateFactomChain()
	err := CommitChain(c)
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Minute)
	err = RevealChain(c)
	if err != nil {
		return err
	}
	return nil
}
*/

func SubmitFactoidTx(m factomwire.Message) error {

	outMsgQueue <- m

	return nil
}
