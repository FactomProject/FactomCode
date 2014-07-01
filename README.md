NotaryChain
===========

NotaryChain Servers begin by providing proof of existence services, but then move on to provide proof of existence of transforms.  A list of such entries can be thought of as a NotaryChain.  A NotaryChain can be used to implement private tokens, smart contracts, smart properties, and more.

NotaryChains leverage the Bitcoin Blockchain, but in a way that minimizes the amount of data actually inserted in the Blockchain.  Thus it provides a mechanism for creating Bitcoin 2.0 services for the trading of assets, securities, commondities, or other complex applications without increasing blockchain "pollution"

State of Development
--------------------

We are very much at an Alpha level of development.  We have a "proof of existence" restful API which runs against the testnet3 Bitcoin network (i.e. not a real thing yet).  There is a client app that uses the "proof of existence" API to register artifacts.


Install
-------

You need to install the following projects.  They each document how to do so, though most are simple "go get" calls.

* https://github.com/conformal/btcd  
* https://github.com/conformal/btcrpcclient
* https://github.com/conformal/btcjson
* https://github.com/conformal/btcwallet
* https://github.com/conformal/btcwire
* https://github.com/conformal/btcwallet
* https://github.com/conformal/btcutil

Then install our projects via:

* go get github.com/NotaryChains/NotaryChainCode/restapi
* go get github.com/NotaryChains/NotaryChainCode/client
