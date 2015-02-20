// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"bytes"
	"encoding/binary"
	"github.com/FactomProject/FactomCode/notaryapi"
)

func (i *Input) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write(i.Txid[:]) //32

	binary.Write(&buf, binary.BigEndian, i.Index) //32

	//count := uint64(len(i.RevealAddr)) // 64
	//binary.Write(&buf, binary.BigEndian, count)
	data, err = i.RevealAddr.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	return buf.Bytes(), err
}

func (i *Input) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 32 + 4 + i.RevealAddr.MarshalledSize()
	return size
}

func (i *Input) UnmarshalBinary(data []byte) (err error) {
	copy(i.Txid[:], data[:32])
	data = data[32:]

	i.Index = binary.BigEndian.Uint32(data[:4])
	data = data[4:]

	i.RevealAddr.UnmarshalBinary(data)

	return nil
}

func (o *Output) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.WriteByte(o.Type) //1

	binary.Write(&buf, binary.BigEndian, o.Amount) //8

	data, err = notaryapi.ByteArray(o.ToAddr).MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	return buf.Bytes(), err
}

func (o *Output) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 1 + 8 + notaryapi.ByteArray(o.ToAddr).MarshalledSize()
	return size
}

func (o *Output) UnmarshalBinary(data []byte) (err error) {
	o.Type = data[0]
	data = data[1:]

	buf := bytes.NewReader(data[:8])
	binary.Read(buf, binary.BigEndian, &o.Amount)
	data = data[8:]

	notaryapi.ByteArray(o.ToAddr).UnmarshalBinary(data)

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
	}

	//Outputs
	binary.Write(&buf, binary.BigEndian, uint64(len(tx.Outputs)))
	for _, out := range tx.Outputs {
		data, err = out.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	binary.Write(&buf, binary.BigEndian, uint32(tx.LockTime))

	return buf.Bytes(), err
}

func (tx *TxData) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 8
	for _, in := range tx.Inputs {
		size += in.MarshalledSize()
	}

	size += 8
	for _, out := range tx.Outputs {
		size += out.MarshalledSize()
	}

	size += 4
	return size
}

func (tx *TxData) UnmarshalBinary(data []byte) (err error) {
	count := binary.BigEndian.Uint64(data[0:8])
	data = data[8:]
	tx.Inputs = make([]Input, count)
	for i := uint64(0); i < count; i++ {
		tx.Inputs[i].UnmarshalBinary(data)
		data = data[tx.Inputs[i].MarshalledSize():]
	}

	count = binary.BigEndian.Uint64(data[0:8])
	data = data[8:]
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
	txm.TxData = new(TxData)
	txm.TxData.UnmarshalBinary(data)
	data = data[txm.TxData.MarshalledSize():]

	count := binary.BigEndian.Uint64(data[0:8])
	data = data[8:]
	txm.Sigs = make([]InputSig, count)
	for i := uint64(0); i < count; i++ {
		txm.Sigs[i].UnmarshalBinary(data)
		data = data[txm.Sigs[i].MarshalledSize():]
	}

	return nil
}
