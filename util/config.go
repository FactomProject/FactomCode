package util

import (
	"log"
	"os"
	"os/user"
	"sync"

	"gopkg.in/gcfg.v1"
)

type FactomdConfig struct {
	App struct {
		PortNumber              int
		HomeDir					string 
		LdbPath                 string 
		BoltDBPath              string 
		DataStorePath           string 
		DirectoryBlockInSeconds int
		NodeMode                string
		ServerPrivKey           string
		ExchangeRate            uint64
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
		PortNumber      int
		ApplicationName string
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
		FactomdAddress   string
		FactomdPort      int
	}
    	Controlpanel struct {
        	Port            string
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
PortNumber				      		= 8088
HomeDir								= ""
LdbPath					        	= "ldb"
BoltDBPath							= ""
DataStorePath			      		= "data/export/"
DirectoryBlockInSeconds				= 60
; --------------- NodeMode: FULL | SERVER | LIGHT ----------------
NodeMode				        	= FULL
ServerPrivKey			      		= 07c0d52cb74f4ca3106d80c4a70488426886bccc6ebc10c6bafb37bf8a65f4c38cee85c62a9e48039d4ac294da97943c2001be1539809ea5f54721f0c5477a0a
ExchangeRate                        = 00666600

[anchor]
ServerECKey							= 397c49e182caa97737c6b394591c614156fbe7998d7bf5d76273961e9fa1edd406ed9e69bfdf85db8aa69820f348d096985bc0b11cc9fc9dcee3b8c68b41dfd5
AnchorChainID						= df3ade9eec4b08d5379cc64270c30ea7315d8a8a1a69efe2b98a60ecdd69e604
ConfirmationsNeeded					= 20

[btc]
WalletPassphrase 	  				= "lindasilva"
CertHomePath			  			= "btcwallet"
RpcClientHost			  			= "localhost:18332"
RpcClientEndpoint					= "ws"
RpcClientUser			  			= "testuser"
RpcClientPass 						= "notarychain"
BtcTransFee				  			= 0.0001
CertHomePathBtcd					= "btcd"
RpcBtcdHost 			  			= "localhost:18334"
RpcUser								=testuser
RpcPass								=notarychain

[wsapi]
ApplicationName						= "Factom/wsapi"
PortNumber				  			= 8088

; ------------------------------------------------------------------------------
; logLevel - allowed values are: debug, info, notice, warning, error, critical, alert, emergency and none
; ------------------------------------------------------------------------------
[log]
logLevel 							= debug
LogPath								= "factom-d.log"

; ------------------------------------------------------------------------------
; Configurations for fctwallet
; ------------------------------------------------------------------------------
[Wallet]
Address          					= localhost
Port             					= 8089
DataFile         					= fctwallet.dat
RefreshInSeconds 					= 60
BoltDBPath 							= ""
FactomdAddress                      = localhost
FactomdPort                         = 8088

; ------------------------------------------------------------------------------
; Configurations for controlpanel
; ------------------------------------------------------------------------------
[Controlpanel]
Port             					= 8090
`

var cfg *FactomdConfig
var once sync.Once
var filename = getHomeDir() + "/.factom/factomd.conf"

// GetConfig reads the default factomd.conf file and returns a FactomConfig
// object corresponding to the state of the file.
func ReadConfig() *FactomdConfig {
	once.Do(func() {
		log.Println("read factom config file: ", filename)
		cfg = readConfig()
	})
	return cfg
}

func ReReadConfig() *FactomdConfig {
	cfg = readConfig()
		
	return cfg
}

func readConfig() *FactomdConfig {
	cfg := new(FactomdConfig)

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
		log.Println("ERROR Reading config file!\nServer starting with default settings...\n",err)
		gcfg.ReadStringInto(cfg, defaultConfig)
	} 
	
	// Default to home directory if not set
	if len(cfg.App.HomeDir) < 1 {
		cfg.App.HomeDir = getHomeDir() + "/.factom/"
	}
	
	// TODO: improve the paths after milestone 1
	cfg.App.LdbPath = cfg.App.HomeDir + cfg.App.LdbPath
	cfg.App.BoltDBPath = cfg.App.HomeDir + cfg.App.BoltDBPath	
	cfg.App.DataStorePath = cfg.App.HomeDir + cfg.App.DataStorePath
	cfg.Log.LogPath = cfg.App.HomeDir + cfg.Log.LogPath
	cfg.Wallet.BoltDBPath = cfg.App.HomeDir + cfg.Wallet.BoltDBPath		
	
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
