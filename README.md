Factom
===========

Factom is an Open-Source project that provides a way to build applications on the Bitcoin blockchain. 

Factom began by providing proof of existence services, but then moved on to provide proof of existence of transforms.  A list of such entries can be thought of as a Factom Chain.  Factom can be used to implement private tokens, smart contracts, smart properties, and more.

Factom leverages the Bitcoin Blockchain, but in a way that minimizes the amount of data actually inserted in the Blockchain.  Thus it provides a mechanism for creating Bitcoin 2.0 services for the trading of assets, securities, commodities, or other complex applications without increasing blockchain "pollution".

State of Development
--------------------

We are very much at an Alpha level of development.  We have a demo client and a Factom server prototype which runs against the testnet3 Bitcoin network. Please check the development branch for latest.   


GETTING STARTED
-------------------

You need to set up Go environment and install the following packages:  

* https://github.com/btcsuite/btcd
* https://github.com/btcsuite/btcwallet
* https://github.com/factomproject/factomcode 


After installing and setting up Go, btcd, and btcwallet, the following command will obtain FactomCode, all dependencies, and install them (completing step 3 above):

```bash
$ go get -u github.com/factomproject/factomcode/...
```

If you receive an error mentioning that go "Cannot download... godoc.org uses insecure protocol" and you have verified that godoc.org is the only domain/resource that is insecurely loading (via http rather than https), you can use the "-insecure" flag to bypass the problem, like so:

```bash
$ go get -insecure -u github.com/factomproject/factomcode/...
```

*Note: the "warning: code.google.com is shutting down" warnings which may appear during installation/setup can safely be ignored.* 
