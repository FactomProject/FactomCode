// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)

package anchor

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/btcsuitereleases/btcd/btcjson/v2/btcjson"
	"github.com/btcsuitereleases/btcd/chaincfg"
	"github.com/btcsuitereleases/btcd/txscript"
	"github.com/btcsuitereleases/btcd/wire"
	"github.com/btcsuitereleases/btcrpcclient"
	"github.com/btcsuitereleases/btcutil"
	"github.com/davecgh/go-spew/spew"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/util"
)

var balances []balance // unspent balance & address & its WIF
var cfg *util.FactomdConfig
var dclient, wclient *btcrpcclient.Client
var fee btcutil.Amount // tx fee for written into btc

type balance struct {
	unspentResult btcjson.ListUnspentResult
	address       btcutil.Address
	wif           *btcutil.WIF
}

// SendRawTransactionToBTC is the main function used to anchor factom
// dir block hash to bitcoin blockchain
func SendRawTransactionToBTC(hash []byte, blockHeight uint64) (*wire.ShaHash, error) {
	b := balances[0]
	i := copy(balances, balances[1:])
	balances[i] = b

	msgtx, err := createRawTransaction(b, hash, blockHeight)
	if err != nil {
		return nil, fmt.Errorf("cannot create Raw Transaction: %s", err)
	}

	shaHash, err := sendRawTransaction(msgtx)
	if err != nil {
		return nil, fmt.Errorf("cannot send Raw Transaction: %s", err)
	}
	return shaHash, nil
}

func createRawTransaction(b balance, hash []byte, blockHeight uint64) (*wire.MsgTx, error) {

	msgtx := wire.NewMsgTx()

	if err := addTxOuts(msgtx, b, hash, blockHeight); err != nil {
		return nil, fmt.Errorf("cannot addTxOuts: %s", err)
	}

	if err := addTxIn(msgtx, b); err != nil {
		return nil, fmt.Errorf("cannot addTxIn: %s", err)
	}

	if err := validateMsgTx(msgtx, []btcjson.ListUnspentResult{b.unspentResult}); err != nil {
		return nil, fmt.Errorf("cannot validateMsgTx: %s", err)
	}

	return msgtx, nil
}

func addTxIn(msgtx *wire.MsgTx, b balance) error {
	output := b.unspentResult
	fmt.Printf("unspentResult: %#v\n", output)
	prevTxHash, err := wire.NewShaHashFromStr(output.TxID)
	if err != nil {
		return fmt.Errorf("cannot get sha hash from str: %s", err)
	}

	outPoint := wire.NewOutPoint(prevTxHash, output.Vout)
	msgtx.AddTxIn(wire.NewTxIn(outPoint, nil))

	// OnRedeemingTx
	err = dclient.NotifySpent([]*wire.OutPoint{outPoint})
	if err != nil {
		fmt.Println("NotifySpent err: ", err.Error())
	}

	subscript, err := hex.DecodeString(output.ScriptPubKey)
	if err != nil {
		return fmt.Errorf("cannot decode scriptPubKey: %s", err)
	}

	sigScript, err := txscript.SignatureScript(msgtx, 0, subscript,
		txscript.SigHashAll, b.wif.PrivKey, true) //.ToECDSA(), true)
	if err != nil {
		return fmt.Errorf("cannot create scriptSig: %s", err)
	}
	msgtx.TxIn[0].SignatureScript = sigScript

	return nil

}

func addTxOuts(msgtx *wire.MsgTx, b balance, hash []byte, blockHeight uint64) error {
	anchorHash, err := prependBlockHeight(blockHeight, hash)
	if err != nil {
		fmt.Printf("ScriptBuilder error: %v\n", err)
	}

	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData(anchorHash)

	// latest routine from Conformal btcsuite returns 2 parameters, not 1... not sure what to do for people with the old conformal libraries :(
	opReturn, err := builder.Script()
	msgtx.AddTxOut(wire.NewTxOut(0, opReturn))
	if err != nil {
		fmt.Printf("ScriptBuilder error: %v\n", err)
	}

	amount, _ := btcutil.NewAmount(b.unspentResult.Amount)
	change := amount - fee

	// Check if there are leftover unspent outputs, and return coins back to
	// a new address we own.
	if change > 0 {

		// Spend change.
		pkScript, err := txscript.PayToAddrScript(b.address)
		if err != nil {
			return fmt.Errorf("cannot create txout script: %s", err)
		}
		msgtx.AddTxOut(wire.NewTxOut(int64(change), pkScript))
	}
	return nil
}

func selectInputs(eligible []btcjson.ListUnspentResult, minconf int) (selected []btcjson.ListUnspentResult, out btcutil.Amount, err error) {
	// Iterate throguh eligible transactions, appending to outputs and
	// increasing out.  This is finished when out is greater than the
	// requested amt to spend.
	selected = make([]btcjson.ListUnspentResult, 0, len(eligible))
	for _, e := range eligible {
		amount, err := btcutil.NewAmount(e.Amount)
		if err != nil {
			fmt.Println("err in creating NewAmount")
			continue
		}
		selected = append(selected, e)
		out += amount
		if out >= fee {
			return selected, out, nil
		}
	}
	if out < fee {
		return nil, 0, fmt.Errorf("insufficient funds: transaction requires %v fee, but only %v spendable", fee, out)
	}

	return selected, out, nil
}

func validateMsgTx(msgtx *wire.MsgTx, inputs []btcjson.ListUnspentResult) error {
	flags := txscript.ScriptBip16 | txscript.ScriptStrictMultiSig //ScriptCanonicalSignatures
	bip16 := time.Now().After(txscript.Bip16Activation)
	if bip16 {
		flags |= txscript.ScriptBip16
	}
	for i, txin := range msgtx.TxIn {

		subscript, err := hex.DecodeString(inputs[i].ScriptPubKey)
		if err != nil {
			return fmt.Errorf("cannot decode scriptPubKey: %s", err)
		}

		engine, err := txscript.NewScript(txin.SignatureScript, subscript, i, msgtx, flags)
		//engine, err := txscript.NewEngine([]byte(inputs[0].ScriptPubKey), msgtx, 0, flags)
		if err != nil {
			return fmt.Errorf("cannot create script engine: %s", err)
		}
		if err = engine.Execute(); err != nil {
			return fmt.Errorf("cannot validate transaction: %s", err)
		}
	}
	return nil

}

func sendRawTransaction(msgtx *wire.MsgTx) (*wire.ShaHash, error) {

	buf := bytes.Buffer{}
	buf.Grow(msgtx.SerializeSize())
	if err := msgtx.BtcEncode(&buf, wire.ProtocolVersion); err != nil {
		// Hitting OOM by growing or writing to a bytes.Buffer already
		// panics, and all returned errors are unexpected.
		//panic(err) //?? should we have retry logic?
		return nil, err
	}

	// use rpc client for btcd here for better callback info
	// this should not require wallet to be unlocked
	shaHash, err := dclient.SendRawTransaction(msgtx, false)
	if err != nil {
		return nil, fmt.Errorf("failed in rpcclient.SendRawTransaction: %s", err)
	}
	fmt.Println("btc txHash returned: ", shaHash) // new tx hash

	return shaHash, nil
}

func createBtcwalletNotificationHandlers() btcrpcclient.NotificationHandlers {

	ntfnHandlers := btcrpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
			//go newBalance(account, balance, confirmed)
			fmt.Println("wclient: OnAccountBalance, account=", account, ", balance=",
				balance.ToUnit(btcutil.AmountBTC), ", confirmed=", confirmed)
		},

		OnWalletLockState: func(locked bool) {
			fmt.Println("wclient: OnWalletLockState, locked=", locked)
		},

		OnUnknownNotification: func(method string, params []json.RawMessage) {
			fmt.Println("wclient: OnUnknownNotification: method=", method, "\nparams[0]=",
				string(params[0]), "\nparam[1]=", string(params[1]))
		},
	}

	return ntfnHandlers
}

func createBtcdNotificationHandlers() btcrpcclient.NotificationHandlers {

	ntfnHandlers := btcrpcclient.NotificationHandlers{

		OnBlockConnected: func(hash *wire.ShaHash, height int32) {
			fmt.Println("dclient: OnBlockConnected: hash=", hash, ", height=", height)
			//go newBlock(hash, height)	// no need
		},

		OnRecvTx: func(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
			//fmt.Printf("dclient: OnRecvTx: details=%#v\n", details)
			//fmt.Printf("dclient: OnRecvTx: tx=%#v,  tx.Sha=%#v, tx.index=%d\n",
			//transaction, transaction.Sha().String(), transaction.Index())
		},

		OnRedeemingTx: func(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
			fmt.Printf("dclient: OnRedeemingTx: details=%#v\n", details)
			fmt.Printf("dclient: OnRedeemingTx: tx.Sha=%#v,  tx.index=%d\n",
				transaction.Sha().String(), transaction.Index())

			if details != nil {
				// do not block OnRedeemingTx callback
				//go saveDirBlockInfo(transaction, details)
				fmt.Println("used to saveDirBlockInfo here.")
			}
		},
	}

	return ntfnHandlers
}

func init() {
	err := initRPCClient()
	if err != nil {
		fmt.Println(err.Error())
	}
	if err := initWallet(); err != nil {
		fmt.Println(err.Error())
	}
}

func initRPCClient() error {
	cfg = util.ReadConfig()
	certHomePath := cfg.Btc.CertHomePath
	rpcClientHost := cfg.Btc.RpcClientHost
	rpcClientEndpoint := cfg.Btc.RpcClientEndpoint
	rpcClientUser := cfg.Btc.RpcClientUser
	rpcClientPass := cfg.Btc.RpcClientPass
	certHomePathBtcd := cfg.Btc.CertHomePathBtcd
	rpcBtcdHost := cfg.Btc.RpcBtcdHost

	// Connect to local btcwallet RPC server using websockets.
	ntfnHandlers := createBtcwalletNotificationHandlers()
	certHomeDir := btcutil.AppDataDir(certHomePath, false)
	certs, err := ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return fmt.Errorf("cannot read rpc.cert file: %s", err)
	}
	connCfg := &btcrpcclient.ConnConfig{
		Host:         rpcClientHost,
		Endpoint:     rpcClientEndpoint,
		User:         rpcClientUser,
		Pass:         rpcClientPass,
		Certificates: certs,
	}
	wclient, err = btcrpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		return fmt.Errorf("cannot create rpc client for btcwallet: %s", err)
	}

	// Connect to local btcd RPC server using websockets.
	dntfnHandlers := createBtcdNotificationHandlers()
	certHomeDir = btcutil.AppDataDir(certHomePathBtcd, false)
	certs, err = ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return fmt.Errorf("cannot read rpc.cert file for btcd rpc server: %s", err)
	}
	dconnCfg := &btcrpcclient.ConnConfig{
		Host:         rpcBtcdHost,
		Endpoint:     rpcClientEndpoint,
		User:         rpcClientUser,
		Pass:         rpcClientPass,
		Certificates: certs,
	}
	dclient, err = btcrpcclient.New(dconnCfg, &dntfnHandlers)
	if err != nil {
		return fmt.Errorf("cannot create rpc client for btcd: %s", err)
	}

	return nil
}

func unlockWallet(timeoutSecs int64) error {
	err := wclient.WalletPassphrase(cfg.Btc.WalletPassphrase, int64(timeoutSecs))
	if err != nil {
		return fmt.Errorf("cannot unlock wallet with passphrase: %s", err)
	}
	return nil
}

func initWallet() error {
	fee, _ = btcutil.NewAmount(cfg.Btc.BtcTransFee)
	err := unlockWallet(int64(600))
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	unspentResults, err := wclient.ListUnspent() //minConf=1
	if err != nil {
		return fmt.Errorf("cannot list unspent. %s", err)
	}
	fmt.Println("unspentResults.len=", len(unspentResults))

	balances = make([]balance, 0, 100)
	if len(unspentResults) > 0 {
		var i int
		for _, b := range unspentResults {
			if b.Amount > fee.ToBTC() {
				balances = append(balances, balance{unspentResult: b})
				i++
			}
		}

	}
	fmt.Println("balances.len=", len(balances))

	for i, b := range balances {
		addr, err := btcutil.DecodeAddress(b.unspentResult.Address, &chaincfg.TestNet3Params)
		if err != nil {
			return fmt.Errorf("cannot decode address: %s", err)
		}
		balances[i].address = addr

		wif, err := wclient.DumpPrivKey(addr)
		if err != nil {
			return fmt.Errorf("cannot get WIF: %s", err)
		}
		balances[i].wif = wif

		fmt.Printf("balance[%d]=%s", i, spew.Sdump(balances[i]))
	}

	//registerNotifications()

	time.Sleep(1 * time.Second)

	return nil
}

func registerNotifications() {
	// OnBlockConnected or OnBlockDisconnected
	err := dclient.NotifyBlocks()
	if err != nil {
		fmt.Println("NotifyBlocks err: ", err.Error())
	}

	// OnTxAccepted: not useful since it covers all addresses
	err = dclient.NotifyNewTransactions(false) //verbose is false
	if err != nil {
		fmt.Println("NotifyNewTransactions err: ", err.Error())
	}

	// OnRecvTx
	addresses := make([]btcutil.Address, 0, 30)
	for _, a := range balances {
		addresses = append(addresses, a.address)
	}
	err = dclient.NotifyReceived(addresses)
	if err != nil {
		fmt.Println("NotifyReceived err: ", err.Error())
	}

	// OnRedeemingTx
	//err := dclient.NotifySpent(outpoints)

}

func compareUnspentResult(a, b btcjson.ListUnspentResult) bool {
	if a.TxID == b.TxID && a.Vout == b.Vout && a.Amount == b.Amount && a.Address == b.Address {
		return true
	}
	return false
}

func shutdown() {
	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	log.Println("Client shutdown in 2 seconds...")
	time.AfterFunc(time.Second*2, func() {
		log.Println("Going down...")
		wclient.Shutdown()
		dclient.Shutdown()
	})
	defer log.Println("Shutdown done!")
	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	wclient.WaitForShutdown()
	dclient.WaitForShutdown()
}

func toHash(txHash *wire.ShaHash) *common.Hash {
	h := new(common.Hash)
	h.SetBytes(txHash.Bytes())
	return h
}

func prependBlockHeight(height uint64, hash []byte) ([]byte, error) {

	if (0 == height) || (0xFFFFFFFFFFFF&height != height) {
		return nil, errors.New("bad block height")
	}

	header := []byte{'F', 'a'}

	big := make([]byte, 8)
	binary.BigEndian.PutUint64(big, height)

	newdata := append(big[2:8], hash...)
	newdata = append(header, newdata...)

	return newdata, nil
}
