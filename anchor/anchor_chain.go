// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)

package anchor

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wire"
)

//Construct the entry and submit it to the server
func submitEntryToAnchorChain(aRecord *AnchorRecord) error {
	fmt.Printf("submitEntryToAnchorChain: start. %s, %#v\n", time.Now(), aRecord)
	//Marshal aRecord into json
	jsonARecord, err := json.Marshal(aRecord)
	//anchorLog.Debug("submitEntryToAnchorChain - jsonARecord: ", string(jsonARecord))
	if err != nil {
		anchorLog.Error("submitEntryToAnchorChain: error in json.Marshal(aRecord). " + err.Error())
		return err
	}
	bufARecord := new(bytes.Buffer)
	bufARecord.Write(jsonARecord)
	//Sign the json aRecord with the server key
	aRecordSig := serverPrivKey.Sign(jsonARecord)
	//Encode sig into Hex string
	// bufARecord.Write([]byte(hex.EncodeToString(aRecordSig.Sig[:])))

	//Create a new entry
	entry := common.NewEntry()
	entry.ChainID = anchorChainID
	// anchorLog.Debug("anchorChainID: ", anchorChainID)

	// instead of append signature at the end of anchor record
	// it can be added as the first entry.ExtIDs[0]
	entry.ExtIDs = append(entry.ExtIDs, []byte(aRecordSig.Sig[:]))
	entry.Content = bufARecord.Bytes()
	anchorLog.Debug("entry: ", spew.Sdump(entry))

	buf := new(bytes.Buffer)
	// 1 byte version
	buf.Write([]byte{0})
	// 6 byte milliTimestamp (truncated unix time)
	buf.Write(milliTime())
	// 32 byte Entry Hash
	buf.Write(entry.Hash().Bytes())
	// 1 byte number of entry credits to pay
	binaryEntry, err := entry.MarshalBinary()
	if err != nil {
		anchorLog.Error("submitEntryToAnchorChain: error in entry.MarshalBinary. " + err.Error())
		return err
	}

	anchorLog.Debug("jsonARecord binary entry: ", hex.EncodeToString(binaryEntry))
	if c, err := util.EntryCost(binaryEntry); err != nil {
		anchorLog.Error("submitEntryToAnchorChain: error in util.EntryCost. " + err.Error())
		return err
	} else {
		buf.WriteByte(byte(c))
	}

	tmp := buf.Bytes()
	sig := serverECKey.Sign(tmp)
	buf = bytes.NewBuffer(tmp)
	buf.Write(serverECKey.Pub.Key[:])
	buf.Write(sig.Sig[:])

	commit := common.NewCommitEntry()
	err = commit.UnmarshalBinary(buf.Bytes())
	if err != nil {
		anchorLog.Error("submitEntryToAnchorChain: error in commit.UnmarshalBinary. " + err.Error())
		return err
	}
	anchorLog.Debug("CommitEntry: ", spew.Sdump(commit))

	// create a CommitEntry msg and send it to the local inmsgQ
	cm := wire.NewMsgCommitEntry()
	cm.CommitEntry = commit
	inMsgQ <- cm
	hash, _ := cm.Sha()
	fmt.Printf("submitEntryToAnchorChain: sending MsgCommitEntry: EntryHash=%s, msg.Hash=%s, %s\n", 
		commit.EntryHash.String(), hash.String(), time.Now())

	// create a RevealEntry msg and send it to the local inmsgQ
	rm := wire.NewMsgRevealEntry()
	rm.Entry = entry
	inMsgQ <- rm
	hash, _ = rm.Sha()
	fmt.Printf("submitEntryToAnchorChain: end. sending MsgRevealEntry: msg.Hash=%s, %s\n", 
		hash.String(), time.Now())

	return nil
}

// MilliTime returns a 6 byte slice representing the unix time in milliseconds
func milliTime() (r []byte) {
	buf := new(bytes.Buffer)
	t := time.Now().UnixNano()
	m := t / 1e6
	binary.Write(buf, binary.BigEndian, m)
	return buf.Bytes()[2:]
}
