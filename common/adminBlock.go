// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"	
    "fmt"
)


// Administrative Chain
type AdminChain struct {
	ChainID         *Hash
	NextBlock       *AdminBlock
	NextBlockHeight  uint32
}


// Administrative Block
// This is a special block which accompanies this Directory Block. 
// It contains the signatures and organizational data needed to validate previous and future Directory Blocks. 
// This block is included in the DB body. It appears there with a pair of the Admin ChainID:SHA256 of the block.
// For more details, please go to:
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#administrative-block
type AdminBlock struct {
	//Marshalized
    ChainID    *Hash        // The Admin ChainID is predefined,
    PrevHash3  *Hash        // This is a SHA3-256 checksum of the previous Admin Block.
    DBHeight   uint32       // Height of the Directory Block associated with this Admin Block
    MsgCount   uint32       // This is the number of Admin Messages and time delimiters contained in this block.
    BodySize   uint32       // Size in bytes of the body
	Msgs       []Msg        // A series of variable sized objects and timestamps arranged in chronological order.
	                        //  Each object is prepended with an AdminID byte.

	//Not Marshalized
	ABHash     *Hash
	MerkleRoot *Hash
	Chain      *AdminChain
}

// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#adminid-bytes
type Msg struct {
    cmd         byte        // See the documentation 
    operands    []byte      // Variable data for each Msg
}

// Defines a set of opcodes and sizes for their operand(s) 
var msg = map[string] []byte { 
    "MinuteNumber"          : []byte{ 0, 1   },     // The preceding data was acknowledged before the minute specified.
    "DBSignature"           : []byte{ 1, 128 },     // The signature of the preceding Directory Block header. 
    "RevealMHash"           : []byte{ 2, 64  },     // The latest M-hash reveal to be considered for determining server priority in subsequent blocks.
    "AddMHash"              : []byte{ 3, 64  },     // Adds or replaces the current M-hash for the specified identity with this new M-hash.
    "IncFedServer"          : []byte{ 4, 1   },     // The server count is incremented by the amount encoded in a following single byte.
    "AddFedServer"          : []byte{ 5, 32  },     // The ChainID of the Federated server which is added to the pool.
    "RemoveFedServer"       : []byte{ 6, 32  },     // The ChainID of the Federated server which is removed from the pool.
    "AddFedServerKey"       : []byte{ 7, 65  },     // Adds an Ed25519 public key to the authority set.
    "AddFedServerBTCKey"    : []byte{ 8, 66  },     // Adds a Bitcoin public key hash to the authority set.
}

// This map gives us a maping of an opcode to its operand size. I could fill out the table
// but I'd rather build it from the msg map, so we don't have to worry about two independent
// definitions.
var msgSize map[byte]int

// ONLY GET the msgSize using THIS function!  That's because we look to see if we have a map yet, and build
// it iff we don't have one yet.
func getMsgSize() map[byte]int {
    if (msgSize == nil){
        msgSize = make((map[byte]int))
        for k := range msg {
            op := msg[k][0]
            msgSize[op] = int(msg[k][1])
        }
    }
    return msgSize
}

// Given a name, this routine returns a new msg struct. This routine will check that the operand is the right length.
func getMsg(name string, operands []byte ) (m *Msg, err error) {
    entry := msg[name]
    m = new(Msg)
    m.cmd = entry[0]
    if len(operands)!= int(entry[1]) {
        return nil, fmt.Errorf("Operand is the incorrect size for ",name)
    }
    m.operands = operands
    return m, nil
}    
    
func CreateAdminBlock(chain *AdminChain, prev *AdminBlock) (b *AdminBlock, err error) {

	b = new(AdminBlock)

	b.ChainID = new(Hash)
    b.ChainID.Bytes = ADMIN_CHAINID

	if prev == nil {
		b.PrevHash3 = NewHash()
	} else {

		if prev.ABHash == nil {
			prev.BuildABHash()
		}
		b.PrevHash3 = prev.ABHash
	}

	b.DBHeight = chain.NextBlockHeight
	b.Msgs     = make([]Msg, 0, AB_CAP)

	return b, err
}

// Just do a simple hash of the whole block.  
func (b *AdminBlock) BuildABHash() (err error) {

	binaryAB, _ := b.MarshalBinary()
	b.ABHash = Sha(binaryAB)

	return
}

//Add a msg to the block, increment our counter!
func (b *AdminBlock) AddABMsg(e Msg) (err error) {
    b.MsgCount++
	b.Msgs = append(b.Msgs, e)
	return
}

// Write out the AdminBlock to binary...
func (b *AdminBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
    
    buf.Write(b.ChainID.Bytes)
    buf.Write(b.PrevHash3.Bytes)

    binary.Write(&buf, binary.BigEndian, b.DBHeight)
    binary.Write(&buf, binary.BigEndian, b.MsgCount)
    binary.Write(&buf, binary.BigEndian, b.BodySize)
    
    for i := uint32(0); i < b.MsgCount; i++ {
        buf.WriteByte(b.Msgs[i].cmd)    
        buf.Write(b.Msgs[i].operands)
	}
	return buf.Bytes(), err
}

// Read in the binary into the Admin block.  I am slicing the data... 
// Maybe we should copy?
func (b *AdminBlock) UnmarshalBinary(data []byte) (err error) {
    
    b.ChainID,  data, err = UnmarshalHash(data)
    b.PrevHash3,data, err = UnmarshalHash(data)

    b.DBHeight, data = binary.BigEndian.Uint32(data[0:4]), data[4:]
    b.MsgCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]
    b.BodySize, data = binary.BigEndian.Uint32(data[0:4]), data[4:]
    
    msm := getMsgSize()
    
	b.Msgs = make([]Msg, b.MsgCount)
	for i := uint32(0); i < b.MsgCount; i++ {
	    b.Msgs[i].cmd,data = data[0], data[1:]
        b.Msgs[i].operands,data = data[:msm[b.Msgs[i].cmd]], data[msm[b.Msgs[i].cmd]:]
	}

	return nil
}



