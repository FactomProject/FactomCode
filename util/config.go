package util

import ()

type FactomdConfig struct {
	App struct {
		PortNumber              int
		LdbPath                 string
		DataStorePath           string
		DirectoryBlockInSeconds int
		NodeMode                string
		FederatedId				string
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
