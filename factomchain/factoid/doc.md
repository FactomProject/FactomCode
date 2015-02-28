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
mywaletaddress := wallet.FactoidAddress()
address, _, _ := factoid.DecodeAddress(mywalletaddress)
```
3) create a TxMsg - a factoid transaction to add 1000 Snow to your wallet address 
```
txmsg := factoid.NewTxFromInputToAddr(myinput, 1000, address)
```
http://godoc.org/github.com/jaybny/FactomCode/factomchain/factoid#NewTxFromInputToAddr

4) use wallet to create a signature of the transaction. (TxMsg.TxData)
```
signature := wallet.DetachMarshalSign(txm.TxData)
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
```

