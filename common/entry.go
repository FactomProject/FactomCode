// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// An Entry is the element which carries user data
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#entry
type Entry struct {
	Version     uint8  	// 1
	ChainID     *Hash   // 33
	ExIDSize    uint16 	// 2
	PayloadSize uint16 	// 2 Total of 38 bytes // to be changed to 37??
	ExtIDs      [][]byte
	Data        []byte
}

func (e *Entry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// Write Version
	binary.Write(&buf, binary.BigEndian, e.Version)

	// Write ChainID
	data, err := e.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	// First compute the ExIDSize 
	var exIDSize uint16

	for _, exId := range e.ExtIDs {
		exIDSize += 2 // Add 2 for the length
		exIDSize += uint16(len(exId))
	}

	// Write ExIDSize
	binary.Write(&buf, binary.BigEndian, exIDSize)

	// Write the Payload Size	
	var payloadsize uint16
	payloadsize = uint16(len(e.Data)) + exIDSize
	if payloadsize > MAX_ENTRY_SIZE {
		return nil, fmt.Errorf("Size of entry exceeds Entry Size Limit, i.e ", payloadsize, " > ", MAX_ENTRY_SIZE)
	}
	binary.Write(&buf, binary.BigEndian, payloadsize)

	// Write out the External IDs
	for _, exId := range e.ExtIDs {
		var size uint16
		size = uint16(len(exId))
		binary.Write(&buf, binary.BigEndian, size)
		buf.Write(exId)
	}

	// Write out the Data
	buf.Write(e.Data)

	return buf.Bytes(), nil
}

func (e *Entry) NewUnmarshalBinary(d []byte) (err error) {
	buf := bytes.NewBuffer(d)
	
	// 1 byte Version
	e.Version, err = buf.ReadByte()
	if err != nil {
		return err
	}
	
	// 32 byte ChainID
	e.ChainID = new(Hash)
	e.ChainID.Bytes = make([]byte, 0, 32)
	if _, err := buf.Read(e.ChainID.Bytes); err != nil {
		return err
	}
	
	// 2 byte size of ExtIDs
	if err := binary.Read(buf, binary.BigEndian, int16(e.ExIDSize)); err != nil {
		return err
	}
	
	// 2 byte size of the Payload
	if err := binary.Read(buf, binary.BigEndian, int16(e.PayloadSize)); err != nil {
		return err
	}
	
	// unmarshal the extids
	for i := e.ExIDSize; i > 0; {
		xsize := int16(0)
		binary.Read(buf, binary.BigEndian, xsize)
		i -= 2
		
		x := make([]byte, 0, xsize)
		if n, err := buf.Read(x); err != nil {
			return err
		} else {
			if c := cap(x); n != c {
				return fmt.Errorf("Could not read ExtID: Read %d bytes of %d\n",
					n, c)
			}
			e.ExtIDs = append(e.ExtIDs, x)
			i -= uint16(n)
		}
	}
	
	// content
	e.Data = buf.Bytes()
	
	return nil
}

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
	totalSize := len(data)
	
	// Get the Version byte
	e.Version, data = data[0], data[1:]
	
	// Get the ChainID	
	e.ChainID = new(Hash)
	err = e.ChainID.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	data = data[e.ChainID.MarshalledSize():]
	
	// Get the External ID Size
	e.ExIDSize, data = binary.BigEndian.Uint16(data[0:2]), data[2:]
	e.PayloadSize, data = binary.BigEndian.Uint16(data[0:2]), data[2:]

	if  totalSize> int(MAX_ENTRY_SIZE) || uint16(totalSize) != e.PayloadSize+38 {		// 38 to be changed to 37??
		return fmt.Errorf("Data is too long, or Lengths don't add up")
	} else if e.ExIDSize > e.PayloadSize {		
		return fmt.Errorf("External IDs are longer than the payload size")
	}

	var size, cnt, eid_len uint16

	datas := data
	for size < e.ExIDSize {
		cnt++
		eid_len, datas = binary.BigEndian.Uint16(datas[0:2]), datas[2:]
		size += eid_len + 2		
		if size > e.ExIDSize {
			return fmt.Errorf("Invalid External IDs")
		}
		datas = datas[eid_len:]
	} 
	
	// we only get out of this nice when size == e.ExIDSize.
	// Otherwise we get an error.
	e.ExtIDs = make([][]byte, cnt, cnt)
	for i := uint16(0); i < cnt; i++ {
		eid_len, data = binary.BigEndian.Uint16(data[0:2]), data[2:]
		e.ExtIDs[i] = make([]byte, eid_len, eid_len)
		copy(e.ExtIDs[i], data[0:eid_len])
		data = data[eid_len:]
	}

	data_len := e.PayloadSize - e.ExIDSize
	e.Data = make([]byte, data_len, data_len)
	copy(e.Data, data)

	return nil
}

func GetChainID(chainName [][]byte) (chainID *Hash, err error) {
	byteSlice := make([]byte, 0, 64)

	if len(chainName) == 0 {
		err = fmt.Errorf("Some name is required to create a ChainID")
		return nil, err
	}

	for _, bytes := range chainName {
		byteSlice = append(byteSlice, Sha(bytes).Bytes...)
	}
	chainID = Sha(byteSlice)
	return chainID, nil
}

// To generate a chain id (hash) from a binary array name
// The algorithm is chainID = Sha(Sha(Name[0]) + Sha(Name[1] + ... + Sha(Name[n])
func (b *Entry) GenerateIDFromName() (chainID *Hash, err error) {
	return GetChainID(b.ExtIDs)
}
