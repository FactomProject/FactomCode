package util_test

import (
	"gopkg.in/gcfg.v1"
	"testing"
	. "github.com/FactomProject/FactomCode/util"
)

func TestLoadDefaultConfig(t *testing.T) {
	type testConfig struct {
		Test struct {
			Foo string
			Bar int64
		}
	}

	var testConfigFile string = `
	[Test]
	Foo = "Bla"
	Bar = "-1"
	`

	cfg := new(testConfig)
	gcfg.ReadStringInto(cfg, testConfigFile)
	if cfg.Test.Foo != "Bla" {
		t.Errorf("Wrong variable read - %v", cfg.Test.Foo)
	}
	if cfg.Test.Bar != -1 {
		t.Errorf("Wrong variable read - %v", cfg.Test.Bar)
	}

	var testConfigFile2 string = `
	[Test]
	Foo = "Ble"
	`
	cfg2 := new(testConfig)
	gcfg.ReadStringInto(cfg2, testConfigFile2)
	if cfg2.Test.Foo != "Ble" {
		t.Errorf("Wrong variable read - %v", cfg.Test.Foo)
	}
	if cfg2.Test.Bar != 0 {
		t.Errorf("Wrong variable read - %v", cfg.Test.Bar)
	}

	gcfg.ReadStringInto(cfg, testConfigFile2)
	if cfg.Test.Foo != "Ble" {
		t.Errorf("Wrong variable read - %v", cfg.Test.Foo)
	}
	if cfg.Test.Bar != -1 {
		t.Errorf("Wrong variable read - %v", cfg.Test.Bar)
	}
}


func TestLoadDefaultConfigFull(t *testing.T) {
	var defaultConfig string = `
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
	ServerPubKey                        = "0426a802617848d4d16d87830fc521f4d136bb2d0c352850919c2679f189613a"
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

	var modifiedConfig string = `
	[app]
	NodeMode                                = "MapMap"
	LdbPath                               = ""
	BoltDBPath                            = "Something"
	`

	cfg := new(FactomdConfig)
	gcfg.ReadStringInto(cfg, defaultConfig)
	if cfg.App.NodeMode != "FULL" {
		t.Errorf("Wrong variable read - %v", cfg.App.NodeMode)
	}
	if cfg.App.LdbPath != "ldb" {
		t.Errorf("Wrong variable read - %v", cfg.App.LdbPath)
	}
	if cfg.App.BoltDBPath != "" {
		t.Errorf("Wrong variable read - %v", cfg.App.BoltDBPath)
	}
	if cfg.App.DataStorePath != "data/export/" {
		t.Errorf("Wrong variable read - %v", cfg.App.DataStorePath)
	}

	gcfg.ReadStringInto(cfg, modifiedConfig)
	if cfg.App.NodeMode != "MapMap" {
		t.Errorf("Wrong variable read - %v", cfg.App.NodeMode)
	}
	if cfg.App.LdbPath != "" {
		t.Errorf("Wrong variable read - %v", cfg.App.LdbPath)
	}
	if cfg.App.BoltDBPath != "Something" {
		t.Errorf("Wrong variable read - %v", cfg.App.BoltDBPath)
	}
	if cfg.App.DataStorePath != "data/export/" {
		t.Errorf("Wrong variable read - %v", cfg.App.DataStorePath)
	}
	if cfg.App.ServerPubKey != "0426a802617848d4d16d87830fc521f4d136bb2d0c352850919c2679f189613a" {
		t.Errorf("Wrong variable read - %v", cfg.App.ServerPubKey)
	}
}