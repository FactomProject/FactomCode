package factomapi

import (
	"net/http"
	"net/url"
	"fmt"
	"encoding/hex"	
	"encoding/base64"		
	"sort"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/notaryapi"	
	"github.com/FactomProject/FactomCode/wallet"
	//"github.com/FactomProject/FactomCode/database/ldb"	
	"strconv"		
	"io/ioutil"	
	"bytes"
	"encoding/binary"	
	"time"

)
//to be improved:
var serverAddr = "localhost:8083"	
var db database.Db // database
var	creditsPerChain int32 = 10

// This method will be replaced with a Factoid transaction once we have the factoid implementation in place
func BuyEntryCredit(version uint16, ecPubKey *notaryapi.Hash, from *notaryapi.Hash, value uint64, fee uint64, sig *notaryapi.Signature) error {


	data := url.Values{}
	data.Set("format", "binary")
	data.Set("datatype", "buycredit")
	data.Set("ECPubKey", ecPubKey.String())
	data.Set("factoidbase", strconv.FormatUint(value, 10))
		
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}

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
		toBlockHeight = uint64(len(dBlocks)-1)
	}
	
	return dBlocks[fromBlockHeight:toBlockHeight+1], nil
}


func GetDirectoryBlokByHash(dBlockHash *notaryapi.Hash) (dBlock *notaryapi.DBlock, err error) {

	dBlock, err = db.FetchDBlockByHash(dBlockHash)
	
	return dBlock, err
}

func GetDirectoryBlokByHashStr(dBlockHashBase64 string) (dBlock *notaryapi.DBlock, err error) {
	
	bytes, err := base64.URLEncoding.DecodeString(dBlockHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	dBlockHash := new (notaryapi.Hash)
	dBlockHash.Bytes = bytes
	
	
	dBlock, _ = db.FetchDBlockByHash(dBlockHash)
	
	return dBlock, nil
}

func GetEntryBlokByHashStr(eBlockHashBase64 string) (eBlock *notaryapi.EBlock, err error) {
	bytes, err := base64.URLEncoding.DecodeString(eBlockHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	eBlockHash := new (notaryapi.Hash)
	eBlockHash.Bytes = bytes

	return GetEntryBlokByHash(eBlockHash)
}

func GetEntryBlokByHash(eBlockHash *notaryapi.Hash) (eBlock *notaryapi.EBlock, err error) {

	eBlock, err = db.FetchEBlockByHash(eBlockHash)
	 
	return eBlock, err
}
 
func GetEntryBlokByMRStr(eBlockMRBase64 string) (eBlock *notaryapi.EBlock, err error) {
	bytes, err := base64.URLEncoding.DecodeString(eBlockMRBase64)
		
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	eBlockMR := new (notaryapi.Hash)
	eBlockMR.Bytes = bytes

	return db.FetchEBlockByMR(eBlockMR)
}

func GetEntryByHashStr(entryHashBase64 string) (entry *notaryapi.Entry, err error) {
	bytes, err := base64.URLEncoding.DecodeString(entryHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	entryHash := new (notaryapi.Hash)
	entryHash.Bytes = bytes

	return GetEntryByHash(entryHash)
}

func GetEntryByHash(entrySha *notaryapi.Hash) (entry *notaryapi.Entry, err error) {

	entry, err = db.FetchEntryByHash(entrySha)

	return entry, err
}


// to be removed------------------------------
func SetServerAddr(addr string) error {
	serverAddr = addr
	
	return nil
}

func SetDB(database database.Db) error {
	db = database
	
	return nil
}
//-=-----------------------------------------

// array sorting implementation
type byBlockID []notaryapi.DBlock
func (f byBlockID) Len() int { 
  return len(f) 
} 
func (f byBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
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
	var msg bytes.Buffer
		
	bChain,_ := c.MarshalBinary()
	//chainhash := notaryapi.Sha(bChain)	
	// Calculate the required credits
	credits := int32(binary.Size(bChain)/1000 + 1) + creditsPerChain 		
	
	binaryEntry, _ := c.FirstEntry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)
	
	entryChainIDHash := notaryapi.Sha(append(c.ChainID.Bytes, entryHash.Bytes ...))	
	
	//msg.Write(bChain) // we don't want to REVEAL the whole chain
	//msg.Write(chainhash.Bytes)//we might not need this??
	
	binary.Write(&msg, binary.BigEndian, uint64(time.Now().Unix()))	
	msg.Write(c.ChainID.Bytes)
	msg.Write(entryHash.Bytes)	
	msg.Write(entryChainIDHash.Bytes) 

	binary.Write(&msg, binary.BigEndian, credits)	

	sig := wallet.SignData(msg.Bytes())	
	 
	// Need to put in a msg obj once we have the P2P networkd 
	data := url.Values {}	
	data.Set("datatype", "commitchain")
	data.Set("format", "binary")
	data.Set("data", hex.EncodeToString(msg.Bytes()))
	data.Set("signature", hex.EncodeToString((*sig.Sig)[:]))
	data.Set("pubkey", hex.EncodeToString((*sig.Pub.Key)[:]))	

	server := fmt.Sprintf(`http://%s/v1`, serverAddr)	
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return nil
}

// RevealChain sends a message to the factom network containing the binary
// encoded first entry for a chain to be used by the server to add a new factom
// chain. It will be rejected if a CommitChain was not done.
func RevealChain(c *notaryapi.EChain) error {
	bChain,_ := c.MarshalBinary()	
	
	data := url.Values{
		"datatype": {"revealchain"},
		"format":   {"binary"},
		"data":     {hex.EncodeToString(bChain)},
	}
				
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)		
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	
	
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
	c.FirstEntry, err = NewEntry(str_name,eids,data)
	return
}

// CommitEntry sends a message to the factom network containing a hash of the
// entry to be used to verify the later RevealEntry.
func CommitEntry(e *notaryapi.Entry) error {
	var msg bytes.Buffer

	bEntry,_ := e.MarshalBinary()
	entryHash := notaryapi.Sha(bEntry)	
	// Calculate the required credits
	credits := int32(binary.Size(bEntry)/1000 + 1)		
	
	binary.Write(&msg, binary.BigEndian, uint64(time.Now().Unix()))
	msg.Write(entryHash.Bytes)
	binary.Write(&msg, binary.BigEndian, credits)		

	sig := wallet.SignData(msg.Bytes())
	// msg.Bytes should be a int64 timestamp followed by a binary entry

	data := url.Values{
		"datatype":  {"commitentry"},
		"format":    {"binary"},
		"signature": {hex.EncodeToString((*sig.Sig)[:])},
		"pubkey":	{hex.EncodeToString((*sig.Pub.Key)[:])},
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
func RevealEntry(e *notaryapi.Entry) error {
	bEntry,_ := e.MarshalBinary()	
	data := url.Values{
		"datatype": {"revealentry"},
		"format":   {"binary"},
		"entry":    {hex.EncodeToString(bEntry)},
	}
	
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)	
	_, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
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
		"datatype":  {"commitentry"},
		"format":    {"binary"},
		"signature": {hex.EncodeToString((*sig.Sig)[:])},
		"pubkey":	{hex.EncodeToString((*sig.Pub.Key)[:])},
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