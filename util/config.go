package util

import (

)

type FactomdConfig struct {
		App struct{
			PortNumber	int		
			LdbPath	string
			DataStorePath string
			DirectoryBlockInSeconds int	
			NodeMode string						
	    }
		Btc struct{
			BTCPubAddr string
			SendToBTCinSeconds int		
			WalletPassphrase string	
			CertHomePath string
			RpcClientHost string
			RpcClientEndpoint string
			RpcClientUser string
			RpcClientPass string
			BtcTransFee float64
			CertHomePathBtcd string
			RpcBtcdHost string
	    }	
		Rpc struct{
			PortNumber	int		
			ApplicationName string
			RefreshInSeconds int
	    }
		Log struct{
	    	LogLevel string
		}
    }
