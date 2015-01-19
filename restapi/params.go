/*
 * Copyright (c) 2013, 2014 Conformal Systems LLC <info@conformal.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"github.com/btcsuite/btcnet"
)

var activeNet = testNet3Params

// params is used to group parameters for various networks such as the main
// network and test networks.
type params struct {
	*btcnet.Params
	connect string
	port    string
}

// mainNetParams contains parameters specific to running btcgui and
// btcwallet on the main network (btcwire.MainNet).
var mainNetParams = params{
	Params:  &btcnet.MainNetParams,
	connect: "localhost:8332",
	port:    "8332",
}

// testNet3Params contains parameters specific to running btcgui and
// btcwallet on the test network (version 3) (btcwire.TestNet3).
var testNet3Params = params{
	Params:  &btcnet.TestNet3Params,
	connect: "localhost:18332",
	port:    "18332",
}

// simNetParams contains parameters specific to running btcgui and
// btcwallet on the simulation test network (btcwire.SimNet).
var simNetParams = params{
	Params:  &btcnet.SimNetParams,
	connect: "localhost:18554",
	port:    "18554",
}
