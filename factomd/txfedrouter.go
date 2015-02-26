// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

//func for determining which federated servers are responsible for confirming msg

import (
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/wallet"
)

//which federated server is responsible for confirming this msg?
func WhosResponsible(*notaryapi.HashF) notaryapi.PublicKey {
	return notaryapi.PubKeyFromString(federatedid)
}

//is this federated server  responsible for confirming this msg?
func IsResponsible(h *notaryapi.HashF, pk notaryapi.PublicKey) bool {
	return WhosResponsible(h) == pk
}

//am I a federated server responsible for confirming this msg?
func ImResponsible(h *notaryapi.HashF) bool {
	return IsResponsible(h, wallet.ClientPublicKey())
}

func VerifyResponsible(h *notaryapi.HashF, msg []byte, sig *[64]byte) bool {
	return WhosResponsible(h).Verify(msg, sig)
}

type FedProcessList struct {
	count  uint32
	height uint64
}

func (fp *FedProcessList) Confirm(h *notaryapi.HashF) *factomwire.MsgConfirmation {
	fp.count++
	mc := factomwire.NewMsgConfirmation(fp.height, fp.count)
	mc.Affirmation = *h
	sig := wallet.Sign(h[:])
	mc.Signature = *sig.Sig
	return mc
}
