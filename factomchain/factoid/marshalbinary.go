// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"bytes"
	"encoding/binary"
	"github.com/FactomProject/FactomCode/notaryapi"
	//"fmt"
)

func (i *Input) MarshalBinary() (data []byte, err error) {
	//fmt.Println("xx i.RevealAddr",len(i.RevealAddr))

	var buf bytes.Buffer

	buf.Write(i.Txid[:]) //32

	binary.Write(&buf, binary.BigEndian, i.Index) //32

	/*
		//count := uint64(len(i.RevealAddr)) // 64
		//binary.Write(&buf, binary.BigEndian, count)
		data, err = i.RevealAddr.MarshalBinary()
		if err != nil {
			return nil, err
		}
	*/
	buf.Write(i.RevealAddr[:]) //32
	buf.Write(data)
	//fmt.Println("xx2 i.RevealAddr",len(i.RevealAddr))

	//fmt.Println("xx i.RevealAddr",len(i.RevealAddr))
	//fmt.Println("i.RevealAddr.MarshalledSize()",i.RevealAddr.MarshalledSize())

	return buf.Bytes(), err
}

func (i *Input) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 32 + 4 + 32 //i.RevealAddr.MarshalledSize()
	//fmt.Println("i.RevealAddr",len(i.RevealAddr))
	//fmt.Println("iiii.RevealAddr.MarshalledSize()",i.RevealAddr.MarshalledSize())

	return size
}

func (i *Input) UnmarshalBinary(data []byte) (err error) {
	copy(i.Txid[:], data[:32])
	data = data[32:]

	i.Index = binary.BigEndian.Uint32(data[:4])
	data = data[4:]

	////fmt.Println("ddd %#v",data)

	//_, i.RevealAddr = i.RevealAddr.UnmarshalBinary(data)
	//fmt.Println("bdd i.RevealAddr",len(i.RevealAddr),i.RevealAddr)

	copy(i.RevealAddr[:], data[:32])
	data = data[32:]
	return nil
}

func (o *Output) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.WriteByte(o.Type) //1

	binary.Write(&buf, binary.BigEndian, o.Amount) //8
	//fmt.Println("o.ToAddr",o.ToAddr,len(data))

	binary.Write(&buf, binary.BigEndian, uint64(len(o.ToAddr)))
	/*
		data, err = notaryapi.ByteArray(o.ToAddr).MarshalBinary()
		if err != nil {
			return nil, err
		}
		//fmt.Println("2.ToAddr",o.ToAddr,len(data))
	*/
	buf.Write(o.ToAddr)

	return buf.Bytes(), err
}

func (o *Output) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 1 + 8 + 8 + uint64(len(o.ToAddr)) //notaryapi.ByteArray(o.ToAddr).MarshalledSize()
	//fmt.Println("o.ToAddr",o.ToAddr)

	/////fmt.Println("notaryapi.ByteArray(o.ToAddr).MarshalledSize()",notaryapi.ByteArray(o.ToAddr).MarshalledSize())
	////fmt.Println("o.ToAddr",o.ToAddr)

	return size
}

func (o *Output) UnmarshalBinary(data []byte) (err error) {
	o.Type = data[0]
	data = data[1:]

	buf := bytes.NewReader(data[:8])
	binary.Read(buf, binary.BigEndian, &o.Amount)
	data = data[8:]

	count := binary.BigEndian.Uint64(data[:8])
	data = data[8:]

	o.ToAddr = make([]byte, count)
	copy(o.ToAddr, data[:count])

	return nil
}

func (tx *TxData) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	//Inputs
	binary.Write(&buf, binary.BigEndian, uint64(len(tx.Inputs)))
	for _, in := range tx.Inputs {
		data, err = in.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		//fmt.Println("hey i.RevealAddr.MarshalledSize()",in.RevealAddr.MarshalledSize())
	}

	//fmt.Println("100 %#v",tx.Outputs)

	//Outputs
	binary.Write(&buf, binary.BigEndian, uint64(len(tx.Outputs)))
	for _, out := range tx.Outputs {
		//fmt.Println("xxzz",out.ToAddr)

		data, err = out.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		//fmt.Println("xzzz",out.ToAddr,len(data),out.MarshalledSize())

	}

	binary.Write(&buf, binary.BigEndian, uint32(tx.LockTime))

	return buf.Bytes(), err
}

func (tx *TxData) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 8
	for _, in := range tx.Inputs {
		size += in.MarshalledSize()
		//fmt.Println("in.MarshalledSize()",in.MarshalledSize())
	}

	size += 8
	for _, out := range tx.Outputs {
		size += out.MarshalledSize()
		//fmt.Println("out.MarshalledSize()",out.MarshalledSize())
	}

	size += 4
	return size
}

func (tx *TxData) UnmarshalBinary(data []byte) (err error) {
	count := binary.BigEndian.Uint64(data[0:8])
	//fmt.Println("counti %#v",count)

	data = data[8:]
	tx.Inputs = make([]Input, count)
	for i := uint64(0); i < count; i++ {
		tx.Inputs[i].UnmarshalBinary(data)
		data = data[tx.Inputs[i].MarshalledSize():]
		//fmt.Println("%#v",tx.Inputs[i].String())

	}

	count = binary.BigEndian.Uint64(data[0:8])

	data = data[8:]
	//fmt.Println("counto %#v",count)

	tx.Outputs = make([]Output, count)
	for i := uint64(0); i < count; i++ {
		tx.Outputs[i].UnmarshalBinary(data)
		data = data[tx.Outputs[i].MarshalledSize():]
	}

	tx.LockTime = binary.BigEndian.Uint32(data[:4])

	return nil
}

func (txm *TxMsg) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = txm.TxData.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	//fmt.Println("before sigs",len(data))
	//fmt.Println("before sigs data %d \n %v",data)// txm.TxData.MarshalledSize(), txm.TxData)

	//Sigs
	binary.Write(&buf, binary.BigEndian, uint64(len(txm.Sigs)))
	for _, s := range txm.Sigs {
		data, err = s.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	return buf.Bytes(), err
}

func (txm *TxMsg) MarshalledSize() uint64 {
	var size uint64 = 0
	size += txm.TxData.MarshalledSize()
	size += 8
	for _, s := range txm.Sigs {
		size += s.MarshalledSize()
	}

	return size
}

func (txm *TxMsg) UnmarshalBinary(data []byte) (err error) {
	//fmt.Println("all data \n %v",data)// txm.TxData.MarshalledSize(), txm.TxData)

	//txm = new(TxMsg)
	txm.TxData = new(TxData)
	txm.TxData.UnmarshalBinary(data)
	data = data[txm.TxData.MarshalledSize():]
	//fmt.Println("all data",txm.TxData.MarshalledSize())// txm.TxData.MarshalledSize(), txm.TxData)

	count := binary.BigEndian.Uint64(data[0:8])
	//fmt.Println("countn",count, data[0:8])// txm.TxData.MarshalledSize(), txm.TxData)

	////fmt.Println("data0",data, count, txm.TxData.MarshalledSize(), txm.TxData)

	data = data[8:]
	txm.Sigs = make([]InputSig, count)
	for i := uint64(0); i < count; i++ {
		txm.Sigs[i].UnmarshalBinary(data)
		data = data[txm.Sigs[i].MarshalledSize():]
	}

	return nil
}

func (b *FChain) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.ChainID.MarshalBinary()
	buf.Write(data)

	count := len(b.Name)
	binary.Write(&buf, binary.BigEndian, uint64(count))

	for _, bytes := range b.Name {
		count = len(bytes)
		binary.Write(&buf, binary.BigEndian, uint64(count))
		buf.Write(bytes)
	}

	return buf.Bytes(), err
}

func (b *FChain) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33 //b.ChainID
	size += 8  //Name length
	for _, bytes := range b.Name {
		size += 8
		size += uint64(len(bytes))
	}
	return size
}

func (b *FChain) UnmarshalBinary(data []byte) (err error) {
	b.ChainID = new(notaryapi.Hash)
	b.ChainID.UnmarshalBinary(data[:33])

	data = data[33:]
	count := binary.BigEndian.Uint64(data[0:8])
	data = data[8:]

	b.Name = make([][]byte, count, count)

	for i := uint64(0); i < count; i++ {
		length := binary.BigEndian.Uint64(data[0:8])
		data = data[8:]
		b.Name[i] = data[:length]
		data = data[length:]
	}

	return nil
}

func (b *FBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, b.Height)
	data, err = b.PrevBlockHash.MarshalBinary()
	if err != nil {
		return
	}

	buf.Write(data)
	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	binary.Write(&buf, binary.BigEndian, b.TxCount)

	return buf.Bytes(), err
}

func (b *FBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 8
	size += b.PrevBlockHash.MarshalledSize()
	size += 8
	size += 4

	return size
}

func (b *FBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.Height, data = binary.BigEndian.Uint64(data[0:8]), data[8:]

	b.PrevBlockHash = new(notaryapi.Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]

	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.TimeStamp = int64(timeStamp)

	b.TxCount = binary.BigEndian.Uint32(data[:4])

	return nil
}

func (b *FBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.Header.MarshalBinary()
	buf.Write(data)

	count := uint64(len(b.Transactions))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i++ {
		data, _ := b.Transactions[i].Txm.MarshalBinary()
		buf.Write(data)
	}
	return buf.Bytes(), err
}

func (b *FBlock) MarshalledSize() uint64 {
	var size uint64 = 0

	size += b.Header.MarshalledSize()
	size += 8 // len(Entries) uint64

	for _, tx := range b.Transactions {
		size += tx.Txm.MarshalledSize()
	}

	return size
}

func (b *FBlock) UnmarshalBinary(data []byte) (err error) {
	h := new(FBlockHeader)
	h.UnmarshalBinary(data)
	b.Header = *h

	data = data[h.MarshalledSize():]

	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]

	b.Transactions = make([]Tx, count)
	for i := uint64(0); i < count; i++ {
		txMsg := new(TxMsg)
		err = txMsg.UnmarshalBinary(data)
		if err != nil {
			return
		}
		data = data[txMsg.MarshalledSize():]

		b.Transactions[i] = *(NewTx(txMsg))
	}

	return nil
}
