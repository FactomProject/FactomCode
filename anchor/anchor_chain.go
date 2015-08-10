// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)
/*
package anchor

import (

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/factomapi"	
)

//Construct the entry and submit it to the server
func SubmitEntry(entryContent []byte) error {

	entry := common.NewEntry()	
	entry.ChainID = anchorChainID
	entry.Content = entryContent

	err := factomapi.CommitEntry(entry)
	if err != nil {
		return err
	}
	err := factomapi.RevealEntry(entry)
	if err != nil {
		return err
	}
	
	
	return nil
}*/