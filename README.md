Factom
===========

Factom is an Open-Source project that provides a way to build applications on the Bitcoin blockchain. 

Factom began by providing proof of existence services, but then move on to provide proof of existence of transforms.  A list of such entries can be thought of as a Factom Chain.  Factom can be used to implement private tokens, smart contracts, smart properties, and more.

Factom leverages the Bitcoin Blockchain, but in a way that minimizes the amount of data actually inserted in the Blockchain.  Thus it provides a mechanism for creating Bitcoin 2.0 services for the trading of assets, securities, commodities, or other complex applications without increasing blockchain "pollution".

State of Development
--------------------

We are very much at an Alpha level of development.  We have a demo client and a Factom server prototype which runs against the testnet3 Bitcoin network. Please check the development branch for latest.   


Getting Started
-------------------

You need to set up Go environment and install the following packages:  

```bash
$ go get github.com/btcsuite/btcd/...
$ go get github.com/btcsuite/btcwallet/...
$ go get github.com/factomproject/factomcode/...
```


