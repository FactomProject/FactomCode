package util

import (
	"log"
	"os"
	"os/user"
	"sync"

	"code.google.com/p/gcfg"
)

type FactomdConfig struct {
	App struct {
		PortNumber              int
		LdbPath                 string // should be removed, and default to $defaultDataDir/ldb9
		BoltDBPath              string // should be removed, 		
		DataStorePath           string // should be removed, and default to $defaultDataDir/store
		DirectoryBlockInSeconds int
		NodeMode                string
		ServerPrivKey           string
	}
	Anchor struct {
		ServerECKey         	string
		AnchorChainID         	string		
		ConfirmationsNeeded 	int	
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
		RpcUser            string
		RpcPass            string
	}
	Rpc struct {
		PortNumber       int
		ApplicationName  string
		RefreshInSeconds int
	}
	Wsapi struct {
		PortNumber       int
		ApplicationName  string
	}
	Log struct {
		LogPath  string
		LogLevel string
	}
	Wallet struct {
		Address          string
		Port             int
		DataFile         string
		RefreshInSeconds string
		BoltDBPath       string
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
BoltDBPath				= "/tmp/"
DataStorePath			= "/tmp/store/seed/"
DirectoryBlockInSeconds	= 60
;---- NodeMode - FULL,SERVER,LIGHT -----
NodeMode				= FULL
ServerPrivKey			= ""

[anchor]
ServerECKey						= 54f7875aaa589126b1111d45d5bbd43ba03512d34d0501d6a05c0aee84d0846294bb4165350cdb7b9e4a7b3b2e7b6e110e4176c7714e53561b8818c3c78d1721
AnchorChainID					= df3ade9eec4b08d5379cc64270c30ea7315d8a8a1a69efe2b98a60ecdd69e604
ConfirmationsNeeded				= 20

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
RpcUser					= ""
RpcPass					= ""


[wsapi]
ApplicationName			= "Factom/wsapi"
PortNumber				= 8088

; ------------------------------------------------------------------------------
; LogLevel - debug,info,notice,warning,error,critical,alert,emergency,none
; ------------------------------------------------------------------------------
[log]
LogLevel 				= warning
LogPath					= /tmp/factomd.log

[Wallet]
Address          				= localhost
Port             				= 8089
DataFile         				= /tmp/fctwallet.dat
RefreshInSeconds 				= 60
BoltDBPath 						= /tmp/
`

var cfg *FactomdConfig
var once sync.Once

// GetConfig reads the default factomd.conf file and returns a FactomConfig
// object corresponding to the state of the file.
func ReadConfig() *FactomdConfig {
	once.Do(func() {
		cfg = readConfig()
	})
	return cfg
}

func readConfig() *FactomdConfig {
	cfg := new(FactomdConfig)
	filename := getHomeDir() + "/.factom/factomd.conf"
	log.Println("read factom config file: ", filename)

	// This makes factom config file located at
	//   POSIX (Linux/BSD): ~/.factom/factom.conf
	//   Mac OS: $HOME/Library/Application Support/Factom/factom.conf
	//   Windows: %LOCALAPPDATA%\Factom\factom.conf
	//   Plan 9: $home/factom/factom.conf
	//factomHomeDir := btcutil.AppDataDir("factom", false)
	//defaultConfigFile := filepath.Join(factomHomeDir, "factomd.conf")
	//
	// eventually we need to make data dir as following
	//defaultDataDir   = filepath.Join(factomHomeDir, "data")
	//LdbPath					 = filepath.Join(defaultDataDir, "ldb9")
	//DataStorePath		 = filepath.Join(defaultDataDir, "store/seed/")

	err := gcfg.ReadFileInto(cfg, filename)
	if err != nil {
		log.Println("Server starting with default settings...")
		gcfg.ReadStringInto(cfg, defaultConfig)
	}
	return cfg
}

func getHomeDir() string {
	// Get the OS specific home directory via the Go standard lib.
	var homeDir string
	usr, err := user.Current()
	if err == nil {
		homeDir = usr.HomeDir
	}

	// Fall back to standard HOME environment variable that works
	// for most POSIX OSes if the directory from the Go standard
	// lib failed.
	if err != nil || homeDir == "" {
		homeDir = os.Getenv("HOME")
	}
	return homeDir
}
