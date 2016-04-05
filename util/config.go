package util

import (
	"log"
	"net"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/FactomProject/FactomCode/common"

	"gopkg.in/gcfg.v1"
)

type FactomdConfig struct {
	App struct {
		HomeDir                 string
		LdbPath                 string
		BoltDBPath              string
		DataStorePath           string
		DirectoryBlockInSeconds int
		NodeMode                string
		NodeID                  string
		InitLeader              bool
		ServerPrivKey           string
		ExchangeRate            uint64
	}
	Anchor struct {
		ServerECKey         string
		AnchorChainID       string
		ConfirmationsNeeded int
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
		Port string
	}
	BTCD BTCDConfig

	//	AddPeers     []string `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	//	ConnectPeers []string `long:"connect" description:"Connect only to the specified peers at startup"`

	Proxy          string `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	DisableListen  bool   `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	DisableRPC     bool   `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass is specified"`
	DisableTLS     bool   `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableDNSSeed bool   `long:"nodnsseed" description:"Disable DNS seeding for peers"`
}

// config defines the configuration options for btcd.
//
// See loadConfig for details on the configuration load process.
type BTCDConfig struct {
	ShowVersion   bool          `short:"V" long:"version" description:"Display version information and exit"`
	ConfigFile    string        `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir       string        `short:"b" long:"datadir" description:"Directory to store data"`
	LogDir        string        `long:"logdir" description:"Directory to log output."`
	AddPeers      []string      `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	ConnectPeers  []string      `long:"connect" description:"Connect only to the specified peers at startup"`
	DisableListen bool          `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	Listeners     []string      `long:"listen" description:"Add an interface/port to listen for connections (default all interfaces port: 8108, testnet: 18108)"`
	MaxPeers      int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	BanDuration   time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	//	RPCUser            string        `short:"u" long:"rpcuser" description:"Username for RPC connections"`
	//	RPCPass            string        `short:"P" long:"rpcpass" default-mask:"-" description:"Password for RPC connections"`
	RPCListeners       []string `long:"rpclisten" description:"Add an interface/port to listen for RPC connections (default port: 8384, testnet: 18334)"`
	RPCCert            string   `long:"rpccert" description:"File containing the certificate file"`
	RPCKey             string   `long:"rpckey" description:"File containing the certificate key"`
	RPCMaxClients      int      `long:"rpcmaxclients" description:"Max number of RPC clients for standard connections"`
	RPCMaxWebsockets   int      `long:"rpcmaxwebsockets" description:"Max number of RPC websocket connections"`
	DisableRPC         bool     `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass is specified"`
	DisableTLS         bool     `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableDNSSeed     bool     `long:"nodnsseed" description:"Disable DNS seeding for peers"`
	ExternalIPs        []string `long:"externalip" description:"Add an ip to the list of local addresses we claim to listen on to peers"`
	Proxy              string   `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyUser          string   `long:"proxyuser" description:"Username for proxy server"`
	ProxyPass          string   `long:"proxypass" default-mask:"-" description:"Password for proxy server"`
	OnionProxy         string   `long:"onion" description:"Connect to tor hidden services via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	OnionProxyUser     string   `long:"onionuser" description:"Username for onion proxy server"`
	OnionProxyPass     string   `long:"onionpass" default-mask:"-" description:"Password for onion proxy server"`
	NoOnion            bool     `long:"noonion" description:"Disable connecting to tor hidden services"`
	TestNet3           bool     `long:"testnet" description:"Use the test network"`
	RegressionTest     bool     `long:"devnet" description:"Use the devnet test network"`
	SimNet             bool     `long:"simnet" description:"Use the simulation test network"`
	DisableCheckpoints bool     `long:"nocheckpoints" description:"Disable built-in checkpoints.  Don't do this unless you know what you're doing."`
	DbType             string   `long:"dbtype" description:"Database backend to use for the Block Chain"`
	Profile            string   `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	CPUProfile         string   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	DebugLevel         string   `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	Upnp               bool     `long:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	FreeTxRelayLimit   float64  `long:"limitfreerelay" description:"Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute"`
	Generate           bool     `long:"generate" description:"Generate (mine) bitcoins using the CPU"`
	MiningAddrs        []string `long:"miningaddr" description:"Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set"`
	BlockMinSize       uint32   `long:"blockminsize" description:"Mininum block size in bytes to be used when creating a block"`
	BlockMaxSize       uint32   `long:"blockmaxsize" description:"Maximum block size in bytes to be used when creating a block"`
	BlockPrioritySize  uint32   `long:"blockprioritysize" description:"Size in bytes for high-priority/low-fee transactions when creating a block"`
	GetWorkKeys        []string `long:"getworkkey" description:"DEPRECATED -- Use the --miningaddr option instead"`
	AddrIndex          bool     `long:"addrindex" description:"Build and maintain a full address index. Currently only supported by leveldb."`
	DropAddrIndex      bool     `long:"dropaddrindex" description:"Deletes the address-based transaction index from the database on start up, and the exits."`
	Onionlookup        func(string) ([]net.IP, error) `json:"-"`
	Lookup             func(string) ([]net.IP, error)`json:"-"`
	Oniondial          func(string, string) (net.Conn, error)`json:"-"`
	Dial               func(string, string) (net.Conn, error)`json:"-"`
	//	miningAddrs        []btcutil.Address
}

// defaultConfig
const defaultConfig = `
; ------------------------------------------------------------------------------
; App settings
; ------------------------------------------------------------------------------
[app]
HomeDir								= ""
LdbPath					        	= "ldb"
BoltDBPath							= ""
DataStorePath			      		= "data/export/"
DirectoryBlockInSeconds				= 60
; --------------- NodeMode: FULL | SERVER | LIGHT ----------------
NodeMode				        	= FULL
; NodeID is a hash hex string uniquely identifying this server and MUST be set for a federate server (NodeMode is SERVER)
NodeID										= "SERVER_DEFAULT"
; This server will start as the ONLY leader initially among federate servers if InitLead is true, and all other servers have to be set as false.
InitLeader							= "false"
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

; ------------------------------------------------------------------------------
; Configurations for BTCD
; ------------------------------------------------------------------------------
[BTCD]
DebugLevel 			= "info"
MaxPeers        	= 125
BanDuration     	= 60000000000
RPCMaxClients   	= 10
RPCMaxWebsockets 	= 25
DbType          	= "leveldb"
FreeTxRelayLimit 	= 15.0
BlockMinSize    	= 0
BlockMaxSize     	= 750000
BlockPrioritySize 	= 50000
Generate 			= false
AddrIndex 			= false
`

var cfg *FactomdConfig
var once sync.Once
var filename = getHomeDir() + "/.factom/factomd.conf"

func SetConfigFile(f string) {
	filename = f
}

// GetConfig reads the default factomd.conf file and returns a FactomConfig
// object corresponding to the state of the file.
func ReadConfig() *FactomdConfig {
	once.Do(func() {
		cfg = readConfig()
	})
	//debug.PrintStack()
	return cfg
}

func ReReadConfig() *FactomdConfig {
	cfg = readConfig()

	return cfg
}

func readConfig() *FactomdConfig {
	if len(os.Args) > 1 { //&& strings.Contains(strings.ToLower(os.Args[1]), "factomd.conf") {
		filename = os.Args[1]
	}
	if strings.HasPrefix(filename, "~") {
		filename = getHomeDir() + filename
	}
	cfg := new(FactomdConfig)
	//log.Println("read factom config file: ", filename)


	err := gcfg.ReadStringInto(cfg, defaultConfig)
	if err != nil {
		panic(err)
	}

	err = gcfg.ReadFileInto(cfg, filename)
	if err != nil {
		log.Println("ERROR Reading config file!\nServer starting with default settings...\n", err)
		gcfg.ReadStringInto(cfg, defaultConfig)
	}

	//log.Println("nodeMode=", cfg.App.NodeMode, ", NodeID=", cfg.App.NodeID)
	// should have a version check here
	if cfg.App.NodeMode == common.SERVER_NODE && len(cfg.App.NodeID) == 0 {
		log.Println("ERROR!!! When running as a federate server (NodeMode is SERVER) in milestone II, NodeID must be set")
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
