package util

import (
	"os"
	"code.google.com/p/gcfg"
)

type FactomdConfig struct {
	App struct {
		PortNumber              int
		LdbPath                 string
		DataStorePath           string
		DirectoryBlockInSeconds int
		NodeMode                string
		FederatedId             string
	}
	Btc struct {
		BTCPubAddr         string
		SendToBTCinSeconds int
		WalletPassphrase   string
		CertHomePath       string
		RpcClientHost      string
		RpcClientEndpoint  string
		RpcClientUser      string
		RpcClientPass      string
		BtcTransFee        float64
		CertHomePathBtcd   string
		RpcBtcdHost        string
	}
	Rpc struct {
		PortNumber       int
		ApplicationName  string
		RefreshInSeconds int
	}
	Log struct {
		LogPath  string
		LogLevel string
	}

	//	AddPeers     []string `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	//	ConnectPeers []string `long:"connect" description:"Connect only to the specified peers at startup"`

	Proxy          string `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	DisableListen  bool   `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	DisableRPC     bool   `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass is specified"`
	DisableTLS     bool   `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableDNSSeed bool   `long:"nodnsseed" description:"Disable DNS seeding for peers"`
}

// defaultConfig
const defaultConfig = `
; ------------------------------------------------------------------------------
; App settings
; ------------------------------------------------------------------------------
[app]
PortNumber				= 8088 
LdbPath					= "/tmp/ldb9"
DataStorePath			= "/tmp/store/seed/"
DirectoryBlockInSeconds	= 60
;---- NodeMode - FULL,SERVER,LIGHT -----
NodeMode				= FULL
FederatedId				= "5706d2ebbc0e1dc7fb1df24d0b6fc6f2b3b35bb04ec316c4683c2"

[btc]
BTCPubAddr				= "movaFTARmsaTMk3j71MpX8HtMURpsKhdra"
SendToBTCinSeconds  	= 600
WalletPassphrase 		= "lindasilva"
CertHomePath			= "btcwallet"
RpcClientHost			= "localhost:18332"
RpcClientEndpoint		= "ws"
RpcClientUser			= "testuser"
RpcClientPass 			= "notarychain"
BtcTransFee				= 0.0001
CertHomePathBtcd		= "btcd"
RpcBtcdHost 			= "localhost:18334"

[rpc]
ApplicationName			= "Factom/factomclient"
PortNumber				= 8088 
RefreshInSeconds		= 60

; ------------------------------------------------------------------------------
; LogLevel - debug,info,notice,warning,error,critical,alert,emergency,none
; ------------------------------------------------------------------------------
[log]
LogLevel 				= warning
LogPath					= /tmp/factomd.log
`

// GetConfig reads the default factomd.conf file and returns a FactomConfig
// object corresponding to the state of the file.
func ReadConfig() *FactomdConfig {
	cfg := new(FactomdConfig)
	filename := os.Getenv("HOME")+"/.factom/factomd.conf"
	err := gcfg.ReadFileInto(cfg, filename)
	if err != nil {
		gcfg.ReadStringInto(cfg, defaultConfig)
	}
	return cfg
}