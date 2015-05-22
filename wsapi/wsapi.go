// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wsapi

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strconv"

	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/util"
	"github.com/hoisie/web"
)

const (
	httpOK  = 200
	httpBad = 400
)

var (
	cfg              = util.ReadConfig().Wsapi
	portNumber       = cfg.PortNumber
	applicationName  = cfg.ApplicationName
	dataStorePath    = "/tmp/store/seed/csv"
	refreshInSeconds = cfg.RefreshInSeconds
)

var server = web.NewServer()

func Start(db database.Db, inMsgQ chan wire.FtmInternalMsg) {
	factomapi.SetDB(db)
	factomapi.SetInMsgQueue(inMsgQ)

	wsLog.Debug("Setting Handlers")
	server.Post("/v1/commit-chain/?", handleCommitChain)
	server.Post("/v1/reveal-chain/?", handleRevealChain)
	server.Post("/v1/commit-entry/?", handleCommitEntry)
	server.Post("/v1/reveal-entry/?", handleRevealEntry)
	server.Get("/v1/directory-block-head/?", handleDirectoryBlockHead)
	server.Get("/v1/directory-block-by-keymr/([^/]+)", handleDirectoryBlock)
	server.Get("/v1/entry-block-by-keymr/([^/]+)", handleEntryBlock)
	server.Get("/v1/entry-by-hash/([^/]+)", handleEntry)
	server.Get("/v1/chain-head/([^/]+)", handleChainHead)
	server.Get("/v1/entry-credit-balance/([^/]+)", handleEntryCreditBalance)

	wsLog.Info("Starting server")
	go web.Run("localhost:" + strconv.Itoa(portNumber))
}

func Stop() {
	server.Close()
}

func handleCommitChain(ctx *web.Context) {
	type commitchain struct {
		CommitChainMsg string
	}

	c := new(commitchain)
	if p, err := ioutil.ReadAll(ctx.Request.Body); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := json.Unmarshal(p, c); err != nil {
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}

	commit := common.NewCommitChain()
	if p, err := hex.DecodeString(c.CommitChainMsg); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := commit.UnmarshalBinary(p); err != nil {		
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}
	if err := factomapi.CommitChain(commit); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	}

	ctx.WriteHeader(httpOK)
}

func handleRevealChain(ctx *web.Context) {
	handleRevealEntry(ctx)
}

func handleCommitEntry(ctx *web.Context) {
	type commitentry struct {
		CommitEntryMsg string
	}

	c := new(commitentry)
	if p, err := ioutil.ReadAll(ctx.Request.Body); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := json.Unmarshal(p, c); err != nil {
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}

	commit := common.NewCommitEntry()
	if p, err := hex.DecodeString(c.CommitEntryMsg); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := commit.UnmarshalBinary(p); err != nil {		
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}
	if err := factomapi.CommitEntry(commit); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	}

	ctx.WriteHeader(httpOK)
}

func handleRevealEntry(ctx *web.Context) {
	type revealentry struct {
		Entry string
	}

	e := new(revealentry)
	if p, err := ioutil.ReadAll(ctx.Request.Body); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := json.Unmarshal(p, e); err != nil {
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}

	entry := common.NewEntry()
	if p, err := hex.DecodeString(e.Entry); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		if err := entry.UnmarshalBinary(p); err != nil {
			wsLog.Error(err)
			ctx.WriteHeader(httpBad)
			return
		}
	}

	if err := factomapi.RevealEntry(entry); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	}

	ctx.WriteHeader(httpOK)
}

func handleDirectoryBlockHead(ctx *web.Context) {
	type dbhead struct {
		KeyMR string
	}

	h := new(dbhead)
	if block, err := factomapi.DBlockHead(); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		h.KeyMR = block.KeyMR.String()
	}

	if p, err := json.Marshal(h); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}

func handleDirectoryBlock(ctx *web.Context, keymr string) {
	type eblockaddr struct {
		ChainID string
		KeyMR   string
	}

	type dblock struct {
		Header struct {
			PrevBlockKeyMR string
			SequenceNumber uint32
			TimeStamp      uint64
		}
		EntryBlockList []eblockaddr
	}

	d := new(dblock)
	if block, err := factomapi.DBlockByKeyMR(keymr); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		d.Header.PrevBlockKeyMR = block.Header.PrevKeyMR.String()
		d.Header.SequenceNumber = block.Header.BlockHeight
		d.Header.TimeStamp = block.Header.StartTime
		for _, v := range block.DBEntries {
			l := new(eblockaddr)
			l.ChainID = v.ChainID.String()
			l.KeyMR = v.MerkleRoot.String()
			d.EntryBlockList = append(d.EntryBlockList, *l)
		}
	}

	if p, err := json.Marshal(d); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}

func handleEntryBlock(ctx *web.Context, keymr string) {
	type entryaddr struct {
		EntryHash string
	}

	type eblock struct {
		Header struct {
			BlockSequenceNumber uint32
			ChainID             string
			PrevKeyMR           string
			TimeStamp           uint64
		}
		EntryList []entryaddr
	}

	e := new(eblock)
	if block, err := factomapi.EBlockByKeyMR(keymr); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		e.Header.BlockSequenceNumber = block.Header.EBHeight
		e.Header.ChainID = block.Header.ChainID.String()
		e.Header.PrevKeyMR = block.Header.PrevKeyMR.String()
		e.Header.TimeStamp = block.Header.StartTime
		for _, v := range block.EBEntries {
			l := new(entryaddr)
			l.EntryHash = v.EntryHash.String()
			e.EntryList = append(e.EntryList, *l)
		}
	}

	if p, err := json.Marshal(e); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}

func handleEntry(ctx *web.Context, hash string) {
	type entry struct {
		ChainID string
		Content string
		ExtIDs  []string
	}

	e := new(entry)
	if entry, err := factomapi.EntryByHash(hash); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		e.ChainID = entry.ChainID.String()
		e.Content = hex.EncodeToString(entry.Content)
		for _, v := range entry.ExtIDs {
			e.ExtIDs = append(e.ExtIDs, hex.EncodeToString(v))
		}
	}

	if p, err := json.Marshal(e); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}

func handleChainHead(ctx *web.Context, chainid string) {
	type chead struct {
		EntryBlockKeyMR string
	}

	c := new(chead)
	if block, err := factomapi.ChainHead(chainid); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		c.EntryBlockKeyMR = block.MerkleRoot.String()
	}

	if p, err := json.Marshal(c); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}

func handleEntryCreditBalance(ctx *web.Context, eckey string) {
	type ecbal struct {
		Balance uint32
	}

	b := new(ecbal)
	if bal, err := factomapi.ECBalance(eckey); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		b.Balance = bal
	}

	if p, err := json.Marshal(b); err != nil {
		wsLog.Error(err)
		ctx.WriteHeader(httpBad)
		return
	} else {
		ctx.Write(p)
	}

	ctx.WriteHeader(httpOK)
}
