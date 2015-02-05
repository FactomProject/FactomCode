// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/FactomProject/FactomCode/factomnet"
	"github.com/FactomProject/FactomCode/factomwire"
)

// activeNetParams is a pointer to the parameters specific to the
// currently active bitcoin network.
var activeNetParams = &mainNetParams

// params is used to group parameters for various networks such as the main
// network and test networks.
type params struct {
	*factomnet.Params
	rpcPort  string
	dnsSeeds []string
}

// mainNetParams contains parameters specific to the main network
// (factomwire.MainNet).  NOTE: The RPC port is intentionally different than the
// reference implementation because btcd does not handle wallet requests.  The
// separate wallet process listens on the well-known port and forwards requests
// it does not handle on to btcd.  This approach allows the wallet process
// to emulate the full reference implementation RPC API.
var mainNetParams = params{
	Params:  &factomnet.MainNetParams,
	rpcPort: "8334",
	dnsSeeds: []string{
		//		"seed.bitcoin.sipa.be",
		"m21.duckdns.org",
		"factom-j.duckdns.org",
		//		"factom-p.duckdns.org",
		//		"devtest.factom.org",
	},
}

// testNet3Params contains parameters specific to the test network (version 3)
// (factomwire.TestNet3).  NOTE: The RPC port is intentionally different than the
// reference implementation - see the mainNetParams comment for details.
var testNet3Params = params{
	Params:   &factomnet.TestNet3Params,
	rpcPort:  "18334",
	dnsSeeds: []string{
	// "factom-p.duckdns.org",
	// "m21.duckdns.org",
	//		"devtest.factom.org",
	},
}

// netName returns the name used when referring to a bitcoin network.  At the
// time of writing, btcd currently places blocks for testnet version 3 in the
// data and log directory "testnet", which does not match the Name field of the
// factomnet parameters.  This function can be used to override this directory name
// as "testnet" when the passed active network matches factomwire.TestNet3.
//
// A proper upgrade to move the data and log directories for this network to
// "testnet3" is planned for the future, at which point this function can be
// removed and the network parameter's name used instead.
func netName(netParams *params) string {
	switch netParams.Net {
	case factomwire.TestNet3:
		return "testnet"
	default:
		return netParams.Name
	}
}
