##GoDoc

http://godoc.org/github.com/jaybny/FactomCode/factomchain/factoid

### To Create a factoid transaction:

1) create an Input from the "faucet": 

http://godoc.org/github.com/jaybny/FactomCode/factomchain/factoid#NewFaucetIn
```
myinput := factoid.NewFaucetIn()
```
2) create an Address from your wallet address
```
mywalletaddress := wallet.FactoidAddress()
address, _, _ := factoid.DecodeAddress(mywalletaddress)
```
3) create a TxMsg - a factoid transaction to add 1000 Snow to your wallet address 
```
txmsg := factoid.NewTxFromInputToAddr(myinput, 1000, address)
```
http://godoc.org/github.com/jaybny/FactomCode/factomchain/factoid#NewTxFromInputToAddr

4) use wallet to create a signature of the transaction. (TxMsg.TxData)
```
signature := wallet.DetachMarshalSign(txmsg.TxData)
```
5) create a factoid SingleSignature and attach it to the factoid transaction 
```
singlesig := factoid.NewSingleSignature(signature)
factoid.AddSingleSigToTxMsg(txmsg, singlesig)
```
##### we now have a signed factoid transaction.

6) create a Tx object which is a transaction (TxMsg) and its Transaction Id (Txid)

```
tx := NewTx(txmsg)
```

7) VerifyTX will return true if all signatures are valid for this transaction 

```
ok := factoid.VerifyTx(tx)
```
##### we have created a faucet transaction and verified its signatures 

### To Create UTXO and add your transaction:
1) create Utxo 
```
utxo := factoid.NewUtxo()
```
2) Add above faucet transaction to Utxo 
```
utxo.AddTx(tx) 
```
##### output of 1000 snow from your transaction is now an Unspent Transaction Output 

### To verify that your transaction is in UTXO, lets try to spend it:

1) get outputs from previous transaction 

```
outs := factoid.OutputsTx(tx)
```
2) generate external address to send to:
```
address2, _, _ := factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSx")
```

3) generate "Reveal Address", your public-key needed to spend previous output 
```
revealaddress := factoid.AddressReveal(*wallet.ClientPublicKey().Key)
```

4) call util.NewTxFromOutputToAddr to generate the new transaction
http://godoc.org/github.com/jaybny/FactomCode/factomchain/factoid#NewTxFromOutputToAddr
```
txid := tx.Id() // the txid of output trying to spend
index := uint32(1) //the output index 

txmsg2 := factoid.NewTxFromOutputToAddr(txid, outs, index, revealaddress, address2)
```

5) sign with wallet and add signature to transaction  
```
factoid.AddSingleSigToTxMsg(txmsg2, factoid.NewSingleSignature(wallet.DetachMarshalSign(txmsg2.TxData)))
```

6) validate inputs againts Utxo, if valid verify signatures and add to Utxo 
```
ok = utxo.IsValid(txmsg2.TxData.Inputs)
if ok {
  tx2 := NewTx(txmsg2)
  ok = factoid.VerifyTx(tx2)
  if ok {
    utxo.AddTx(tx2)
  }
}
```
##### We now just sent the 1000 coins we received from the faucet to "ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSx"
