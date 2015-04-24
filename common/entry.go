// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package common

// Defines the Entries

import (
	"bytes"
	"encoding/binary"
    "fmt"
)

type Entry struct {
    Version     uint8   // 1
    ChainID     Hash    // 32
	ExIDSize    uint16  // 2
	PayloadSize uint16  // 2 Total of 37 bytes
	ExtIDs      [][]byte
	Data        []byte
}

func (e Entry) MarshalledSize() uint64 {
    return uint64 ( 8+HASH_LENGTH+2+2+ int(e.PayloadSize))  
}  

func (e *Entry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
    
    // Write Version
    binary.Write(&buf,binary.BigEndian, e.Version)

    // Write ChainID
    {
	    data, err := e.ChainID.MarshalBinary()
	    if err != nil {
		    return nil, err
	    }
        buf.Write(data)
    }
    
    // First compute the ExIDSize (just in case someone edited the ExtIDs
    var exIDSize uint16
    
    for _,exId := range e.ExtIDs {
       exIDSize += 2;  // Add 2 for the length
       exIDSize += uint16(len(exId))
    }
    
    // Write ExIDSize
    binary.Write(&buf,binary.BigEndian,exIDSize)
    
    // Write the Payload Size
    var totalsize uint16
    totalsize = uint16(len(e.Data)) + exIDSize
    if(totalsize > MAX_ENTRY_SIZE){
       return nil,fmt.Errorf("Size of entry exceeds Entry Size Limit, i.e ",totalsize," > ",MAX_ENTRY_SIZE)
    } 
    binary.Write(&buf,binary.BigEndian,totalsize)   
         
    // Write out the External IDs
    for _,exId := range e.ExtIDs {
       var size uint16
       size = uint16(len(exId))
       binary.Write(&buf,binary.BigEndian,size)
       buf.Write(exId)
    }
    
    // Write out the Data
    buf.Write(e.Data)
        
	return buf.Bytes(), nil
	
	data, err := e.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(data)
	if err != nil {
		return nil, err
	}

	buf.Write(e.Data)

	return buf.Bytes(), nil
}

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
    // Get the Version byte
	e.Version, data = data[0], data[1:]
	// Get the ChainID
	err = e.ChainID.UnmarshalBinary(data[:HASH_LENGTH])
	if(err != nil) {
        return err
    }
    // Get the External ID Size
    e.ExIDSize,   data = binary.BigEndian.Uint16(data[0:2]), data[2:]
    e.PayloadSize,data = binary.BigEndian.Uint16(data[0:2]), data[2:]
    
    if len(data) > 10240 || uint16(len(data)) != e.PayloadSize+37 {
       return fmt.Errorf("Data is too long, or Lengths don't add up") 
    } else if e.ExIDSize>e.PayloadSize {
       return fmt.Errorf("External IDs are longer than the payload size")
    }
    
    var size, cnt uint16
    datas := data
    for size < e.ExIDSize {
		cnt++
        eid_len, datas := binary.BigEndian.Uint16(datas[0:2]), datas[2:]
		size += eid_len
		if size > e.ExIDSize {
           return fmt.Errorf("Invalid External IDs")
        }
        datas = datas[eid_len:]
    } // we only get out of this nice when size == e.ExIDSize.  
      // Otherwise we get an error.
    e.ExtIDs = make([][]byte,cnt,cnt)
    for i := uint16(0); i<cnt; i++ {
        eid_len, data := binary.BigEndian.Uint16(data[0:2]), data[2:]
        e.ExtIDs[i] = make([]byte,eid_len,eid_len)
        copy(e.ExtIDs[i],data[0:eid_len])
        data = data[eid_len:]
    }
    
    data_len := e.PayloadSize-e.ExIDSize
    e.Data = make([]byte,data_len,data_len)
    copy(e.Data,data)

	return nil
}


