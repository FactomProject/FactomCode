package main

import (
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"strings"
	"strconv"
	"fmt"
)

func find(path []string) (interface{}, *notaryapi.Error) {
	if len(path) == 0 {
		return nil, notaryapi.CreateError(notaryapi.ErrorMissingVersionSpec, "")
	}
	
	ver, path := path[0], path[1:] // capture version spec
	
	if !strings.HasPrefix(ver, "v") {
		return nil, notaryapi.CreateError(notaryapi.ErrorMalformedVersionSpec, fmt.Sprintf(`The version specifier "%s" is malformed`, ver))
	}
	
	ver = strings.TrimPrefix(ver, "v")
	
	if ver == "1" {
		return findV1("/v1", path)
	}
	
	return nil, notaryapi.CreateError(notaryapi.ErrorBadVersionSpec, fmt.Sprintf(`The version specifier "v%s" does not refer to a supported version`, ver))
}

func findV1(context string, path []string) (interface{}, *notaryapi.Error) {
	if len(path) == 0 {
		return nil, notaryapi.CreateError(notaryapi.ErrorEmptyRequest, "")
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "blocks" {
		return nil, notaryapi.CreateError(notaryapi.ErrorBadElementSpec, fmt.Sprintf(`The element specifier "%s" is not valid in the context "%s"`, root, context))
	}
	
	return findV1InBlocks(context + "/" + root, path, blocks)
}

func findV1InBlocks(context string, path []string, blocks []*notaryapi.Block) (interface{}, *notaryapi.Error) {
	if len(path) == 0 {
		return blocks, nil
	}
	
	// capture root spec
	sid := path[0]
	iid, err := strconv.Atoi(sid)
	id := uint64(iid)
	path = path[1:]
	
	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorBadIdentifier, fmt.Sprintf(`The identifier "%s" is malformed: %s`, sid, err.Error()))
	}
	
	if len(blocks) == 0 {
		return nil, notaryapi.CreateError(notaryapi.ErrorBlockNotFound, fmt.Sprintf(`The no blocks can be found in the context "%s"`, sid, context))
	}
	
	idOffset := blocks[0].BlockID
	
	if id < idOffset {
		return nil, notaryapi.CreateError(notaryapi.ErrorBlockNotFound, fmt.Sprintf(`The block identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	id = id - idOffset
	
	if len(blocks) <= int(id) {
		return nil, notaryapi.CreateError(notaryapi.ErrorBlockNotFound, fmt.Sprintf(`The block identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	return findV1InBlock(context + "/" + sid, path, blocks[id])
}

func findV1InBlock(context string, path []string, block *notaryapi.Block) (interface{}, *notaryapi.Error) {
	if len(path) == 0 {
		return block, nil
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "entries" {
		return nil, nil // bad root spec
	}
	
	return findV1InEntries(context + "/" + root, path, block.Entries)
}

func findV1InEntries(context string, path []string, entries []*notaryapi.Entry) (interface{}, *notaryapi.Error) {
	if len(path) == 0 {
		return entries, nil
	}
	
	// capture root spec
	sid := path[0]
	id, err := strconv.Atoi(path[0])
	path = path[1:]
	
	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorBadIdentifier, fmt.Sprintf(`The identifier "%s" is malformed%s`, sid, err.Error()))
	}
	
	if len(entries) <= id {
		return nil, notaryapi.CreateError(notaryapi.ErrorEntryNotFound, fmt.Sprintf(`The entry identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	return entries[id], nil
}

