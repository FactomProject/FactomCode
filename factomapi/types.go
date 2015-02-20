package factomapi

import (
	"bytes"
	"crypto/sha256"
	//	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
)

type jsonentry struct {
	ChainID string
	ExtIDs  []string
	Data    string
}

// Objects implimenting the FactomWriter interface may be used in the Submit
// call to create and add an entry to the factom network.
type FactomWriter interface {
	CreateFactomEntry() *Entry
}

// Objects implimenting the FactomChainer interface may be used in the
// CreateChain call to create a chain and first entry on the factom network.
type FactomChainer interface {
	CreateFactomChain() *Chain
}

// A factom entry that can be submitted to the factom network.
type Entry struct {
	ChainID []byte
	ExtIDs  [][]byte
	Data    []byte
}

// CreateFactomEntry allows an Entry to satisfy the FactomWriter interface.
func (e *Entry) CreateFactomEntry() *Entry {
	return e
}

// Hash returns a hex encoded sha256 hash of the entry.
func (e *Entry) Hash() string {
	s := sha256.New()
	s.Write(e.MarshalBinary())
	return hex.EncodeToString(s.Sum(nil))
}

// Hex return the hex encoded string of the binary entry.
// Depricated!
func (e *Entry) Hex() string {
	return hex.EncodeToString(e.MarshalBinary())
}

// MarshalBinary creates a single []byte from an entry for transport.
func (e *Entry) MarshalBinary() []byte {
	var buf bytes.Buffer

	buf.Write([]byte{byte(len(e.ChainID))})
	buf.Write(e.ChainID)

	count := len(e.ExtIDs)
	binary.Write(&buf, binary.BigEndian, uint8(count))
	for _, bytes := range e.ExtIDs {
		count = len(bytes)
		binary.Write(&buf, binary.BigEndian, uint32(count))
		buf.Write(bytes)
	}

	buf.Write(e.Data)

	return buf.Bytes()
}

// UnmarshalJSON makes satisfies the json.Unmarshaler interfact and populates
// an entry with the data from a json entry.
func (e *Entry) UnmarshalJSON(b []byte) (err error) {
	var j jsonentry

	json.Unmarshal(b, &j)

	return e.FromJsonEntry(j)
}

func (e *Entry) FromJsonEntry(j jsonentry) (err error) {
	e.ChainID, err = hex.DecodeString(j.ChainID)
	if err != nil {
		return err
	}
	for _, v := range j.ExtIDs {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	e.Data, err = hex.DecodeString(j.Data)
	if err != nil {
		return err
	}

	return nil
}

// UnmarshalJSON makes satisfies the json.Unmarshaler interfact and populates
// an entry with the data from a json entry.
func (e *Chain) UnmarshalJSON(b []byte) (err error) {
	var j jsonchain

	json.Unmarshal(b, &j)

	e.ChainID, err = hex.DecodeString(j.ChainID)
	if err != nil {
		return err
	}
	for _, v := range j.Name {
		e.Name = append(e.Name, []byte(v))
	}

	e.FirstEntry = new(Entry)
	err = e.FirstEntry.FromJsonEntry(j.FirstEntry)
	if err != nil {
		return err
	}

	return nil
}

type jsonchain struct {
	ChainID    string
	Name       []string
	FirstEntry jsonentry
}

// A Chain that can be submitted to the factom network.
type Chain struct {
	ChainID    []byte
	Name       [][]byte
	FirstEntry *Entry
}

// CreateFactomChain satisfies the FactomChainer interface.
func (c *Chain) CreateFactomChain() *Chain {
	return c
}

// GenerateID will create the chainid from the chain name. It sets the chainid
// for the object and returns the chainid as a hex encoded string.
func (c *Chain) GenerateID() string {
	b := make([]byte, 0, 32)
	for _, v := range c.Name {
		for _, w := range sha(v) {
			b = append(b, w)
		}
	}
	c.ChainID = sha(b)
	return hex.EncodeToString(c.ChainID)
}

// Hash will return a hex encoded hash of the chainid, a hash of the entry, and
// a hash of the chainid + entry to be used by CommitChain.
func (c *Chain) Hash() (chain string, chain_entry string, entry string) {
	s := sha256.New()
	s.Write(c.MarshalBinary())
	chain = hex.EncodeToString(s.Sum(nil))

	s.Write(c.FirstEntry.MarshalBinary())
	chain_entry = hex.EncodeToString(s.Sum(nil))

	entry = c.FirstEntry.Hash()

	return
}

// Hex will return a hex encoded string of the binary chain.
func (c *Chain) Hex() string {
	return hex.EncodeToString(c.MarshalBinary())
}

// MarshalBinary creates a single []byte from a chain for transport.
func (c *Chain) MarshalBinary() []byte {
	var buf bytes.Buffer

	buf.Write(c.ChainID)

	count := len(c.Name)
	binary.Write(&buf, binary.BigEndian, uint64(count))

	for _, bytes := range c.Name {
		count = len(bytes)
		binary.Write(&buf, binary.BigEndian, uint64(count))
		buf.Write(bytes)
	}

	return buf.Bytes()
}

func sha(b []byte) []byte {
	s := sha256.New()
	s.Write(b)
	return s.Sum(nil)
}
