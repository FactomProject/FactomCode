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
	"strconv"
	"time"

	"github.com/btcsuitereleases/btcd/btcjson"
	"github.com/btcsuitereleases/btcd/chaincfg"
	"github.com/btcsuitereleases/btcd/txscript"
	"github.com/btcsuitereleases/btcd/wire"
	"github.com/btcsuitereleases/btcrpcclient"
	"github.com/btcsuitereleases/btcutil"
	"github.com/davecgh/go-spew/spew"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
)

var (
	balances         []balance // unspent balance & address & its WIF
	cfg              *util.FactomdConfig
	dclient, wclient *btcrpcclient.Client
	fee              btcutil.Amount                  // tx fee for written into btc
	dirBlockInfoMap  map[string]*common.DirBlockInfo //dbHash string as key
	db               database.Db
	//Server Entry Credit private key
	serverECKey common.PrivateKey	
	//Anchor chain ID
	anchorChainID common.Hash	
	confirmationsNeeded int
)

type balance struct {
	unspentResult btcjson.ListUnspentResult
	address       btcutil.Address
	wif           *btcutil.WIF
}

// SendRawTransactionToBTC is the main function used to anchor factom
// dir block hash to bitcoin blockchain
func SendRawTransactionToBTC(hash *common.Hash, blockHeight uint64) (*wire.ShaHash, error) {
	anchorLog.Debug("SendRawTransactionToBTC: hash=", hash.String(), ", dir block height=", strconv.FormatUint(blockHeight, 10))
	dirBlockInfo, err := sanityCheck(hash)
	if err != nil {
		return nil, err
	}
	return doTransaction(hash, blockHeight, dirBlockInfo)
}

func doTransaction(hash *common.Hash, blockHeight uint64, dirBlockInfo *common.DirBlockInfo) (*wire.ShaHash, error) {
	b := balances[0]
	i := copy(balances, balances[1:])
	balances[i] = b

	msgtx, err := createRawTransaction(b, hash.Bytes(), blockHeight)
	if err != nil {
		return nil, fmt.Errorf("cannot create Raw Transaction: %s", err)
	}

	shaHash, err := sendRawTransaction(msgtx)
	if err != nil {
		return nil, fmt.Errorf("cannot send Raw Transaction: %s", err)
	}
	// for test purpose
	if dirBlockInfo != nil {
		dirBlockInfo.BTCTxHash = toHash(shaHash)
	}
	return shaHash, nil
}

func sanityCheck(hash *common.Hash) (*common.DirBlockInfo, error) {
	dirBlockInfo := dirBlockInfoMap[hash.String()]
	if dirBlockInfo == nil {
		s := fmt.Sprintf("Anchor Error: hash %s does not exist in dirBlockInfoMap.\n", hash.String())
		anchorLog.Error(s)
		return nil, errors.New(s)
	}
	if dirBlockInfo.BTCConfirmed {
		s := fmt.Sprintf("Anchor Warning: hash %s has already been confirmed in btc block chain.\n", hash.String())
		anchorLog.Error(s)
		return nil, errors.New(s)
	}
	if !common.NewHash().IsSameAs(dirBlockInfo.BTCTxHash) {
		s := fmt.Sprintf("Anchor Warning: hash %s has already been anchored but not confirmed. btc tx hash is %s\n", hash.String(), dirBlockInfo.BTCTxHash.String())
		anchorLog.Error(s)
		return nil, errors.New(s)
	}
	if dclient == nil || wclient == nil || balances == nil {
		s := fmt.Sprintf("\n\n$$$ WARNING: rpc clients and/or wallet are not initiated successfully. No anchoring for now.\n")
		anchorLog.Warning(s)
		return nil, errors.New(s)
	}
	return dirBlockInfo, nil
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
	anchorLog.Info("unspentResult: %#v\n", output)
	prevTxHash, err := wire.NewShaHashFromStr(output.TxID)
	if err != nil {
		return fmt.Errorf("cannot get sha hash from str: %s", err)
	}

	outPoint := wire.NewOutPoint(prevTxHash, output.Vout)
	msgtx.AddTxIn(wire.NewTxIn(outPoint, nil))

	// OnRedeemingTx
	err = dclient.NotifySpent([]*wire.OutPoint{outPoint})
	if err != nil {
		anchorLog.Error("NotifySpent err: ", err.Error())
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
		anchorLog.Error("ScriptBuilder error: %v\n", err)
	}

	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData(anchorHash)

	// latest routine from Conformal btcsuite returns 2 parameters, not 1... not sure what to do for people with the old conformal libraries :(
	opReturn, err := builder.Script()
	msgtx.AddTxOut(wire.NewTxOut(0, opReturn))
	if err != nil {
		anchorLog.Error("ScriptBuilder error: %v\n", err)
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
			anchorLog.Error("err in creating NewAmount")
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

	for i := range msgtx.TxIn {
		scriptPubKey, err := hex.DecodeString(inputs[i].ScriptPubKey)
		if err != nil {
			return fmt.Errorf("cannot decode scriptPubKey: %s", err)
		}
		//engine, err := txscript.NewScript(txin.SignatureScript, scriptPubKey, i, msgtx, flags)
		engine, err := txscript.NewEngine(scriptPubKey, msgtx, i, flags)
		if err != nil {
			anchorLog.Error("cannot create script engine: %s\n", err)
			return fmt.Errorf("cannot create script engine: %s", err)
		}
		if err = engine.Execute(); err != nil {
			anchorLog.Error("cannot execute script engine: %s\n", err)
			return fmt.Errorf("cannot validate transaction: %s", err)
		}
	}
	return nil
}

func sendRawTransaction(msgtx *wire.MsgTx) (*wire.ShaHash, error) {
	anchorLog.Debug()
	buf := bytes.Buffer{}
	buf.Grow(msgtx.SerializeSize())
	if err := msgtx.BtcEncode(&buf, wire.ProtocolVersion); err != nil {
		// Hitting OOM by growing or writing to a bytes.Buffer already
		// panics, and all returned errors are unexpected.
		//panic(err)
		//TODO: should we have retry logic?
		return nil, err
	}

	// use rpc client for btcd here for better callback info
	// this should not require wallet to be unlocked
	shaHash, err := dclient.SendRawTransaction(msgtx, false)
	if err != nil {
		return nil, fmt.Errorf("failed in rpcclient.SendRawTransaction: %s", err)
	}
	anchorLog.Info("btc txHash returned: ", shaHash) // new tx hash
	return shaHash, nil
}

func createBtcwalletNotificationHandlers() btcrpcclient.NotificationHandlers {

	ntfnHandlers := btcrpcclient.NotificationHandlers{
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
			//go newBalance(account, balance, confirmed)
			//anchorLog.Info("wclient: OnAccountBalance, account=", account, ", balance=",
			//balance.ToUnit(btcutil.AmountBTC), ", confirmed=", confirmed)
		},

		OnWalletLockState: func(locked bool) {
			//anchorLog.Info("wclient: OnWalletLockState, locked=", locked)
		},

		OnUnknownNotification: func(method string, params []json.RawMessage) {
			//anchorLog.Info("wclient: OnUnknownNotification: method=", method, "\nparams[0]=",
			//string(params[0]), "\nparam[1]=", string(params[1]))
		},
	}

	return ntfnHandlers
}

func createBtcdNotificationHandlers() btcrpcclient.NotificationHandlers {

	ntfnHandlers := btcrpcclient.NotificationHandlers{

		OnBlockConnected: func(hash *wire.ShaHash, height int32) {
			anchorLog.Info("dclient: OnBlockConnected: hash=", hash, ", height=", height)
			//go newBlock(hash, height)	// no need
		},

		OnRecvTx: func(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
			//anchorLog.Info("dclient: OnRecvTx: details=%#v\n", details)
			//anchorLog.Info("dclient: OnRecvTx: tx=%#v,  tx.Sha=%#v, tx.index=%d\n",
			//transaction, transaction.Sha().String(), transaction.Index())
		},

		OnRedeemingTx: func(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
			//anchorLog.Info("dclient: OnRedeemingTx: details=%#v\n", details)
			//anchorLog.Info("dclient: OnRedeemingTx: tx.Sha=%#v,  tx.index=%d\n",
			//transaction.Sha().String(), transaction.Index())

			if details != nil {
				// do not block OnRedeemingTx callback
				anchorLog.Info("Anchor: saveDirBlockInfo.")
				go saveDirBlockInfo(transaction, details)
			}
		},
	}

	return ntfnHandlers
}

// InitAnchor inits rpc clients for factom
// and load up unconfirmed DirBlockInfo from leveldb
func InitAnchor(ldb database.Db, serverECKey common.PrivateKey, anchorChainID common.Hash) {
	anchorLog.Debug("InitAnchor")
	db = ldb
	var err error
	dirBlockInfoMap, err = db.FetchAllUnconfirmedDirBlockInfo()
	if err != nil {
		anchorLog.Error("InitAnchor error - " + err.Error())
		return
	}

	if err = initRPCClient(); err != nil {
		anchorLog.Error(err.Error())
		return
	}
	//defer shutdown(dclient)
	//defer shutdown(wclient)

	if err = initWallet(); err != nil {
		anchorLog.Error(err.Error())
		return
	}
	return
}

func initRPCClient() error {
	anchorLog.Debug("init RPC client")
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
	balances = make([]balance, 0, 100)
	fee, _ = btcutil.NewAmount(cfg.Btc.BtcTransFee)
	err := unlockWallet(int64(600))
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	unspentResults, err := wclient.ListUnspent() //minConf=1
	if err != nil {
		return fmt.Errorf("cannot list unspent. %s", err)
	}
	anchorLog.Info("unspentResults.len=", len(unspentResults))

	if len(unspentResults) > 0 {
		var i int
		for _, b := range unspentResults {
			if b.Amount > fee.ToBTC() {
				balances = append(balances, balance{unspentResult: b})
				i++
			}
		}
	}
	anchorLog.Info("balances.len=", len(balances))

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
		anchorLog.Info("balance[%d]=%s", i, spew.Sdump(balances[i]))
	}

	time.Sleep(1 * time.Second)
	return nil
}

func shutdown(client *btcrpcclient.Client) {
	if client == nil {
		return
	}
	// For this example gracefully shutdown the client after 10 seconds.
	// Ordinarily when to shutdown the client is highly application
	// specific.
	log.Println("Client shutdown in 2 seconds...")
	time.AfterFunc(time.Second*2, func() {
		log.Println("Going down...")
		client.Shutdown()
	})
	defer log.Println("btcsuite client shutdown is done!")
	// Wait until the client either shuts down gracefully (or the user
	// terminates the process with Ctrl+C).
	client.WaitForShutdown()
}

func prependBlockHeight(height uint64, hash []byte) ([]byte, error) {
	// dir block genesis block height starts with 0, for now
	// similar to bitcoin genesis block
	//if (0 == height) || (0xFFFFFFFFFFFF&height != height) {
	if 0xFFFFFFFFFFFF&height != height {
		return nil, errors.New("bad block height")
	}

	header := []byte{'F', 'a'}
	big := make([]byte, 8)
	binary.BigEndian.PutUint64(big, height)

	newdata := append(big[2:8], hash...)
	newdata = append(header, newdata...)
	return newdata, nil
}

func saveDirBlockInfo(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
	anchorLog.Debug("to save dir block hash to btc.")
	var saved = false
	for _, dirBlockInfo := range dirBlockInfoMap {
		if dirBlockInfo.BTCTxHash != nil &&
			bytes.Compare(dirBlockInfo.BTCTxHash.Bytes(), transaction.Sha().Bytes()) == 0 {
			dirBlockInfo.BTCTxOffset = int32(details.Index)
			dirBlockInfo.BTCBlockHeight = details.Height
			dirBlockInfo.BTCConfirmed = true
			db.InsertDirBlockInfo(dirBlockInfo)
			delete(dirBlockInfoMap, dirBlockInfo.DBMerkleRoot.String())
			anchorLog.Info("In saveDirBlockInfo, dirBlockInfo:%+v saved to db\n", dirBlockInfo)
			saved = true
			break
		}
	}
	// should not happen at all?
	if !saved {
		anchorLog.Info("Not saved to db: ")
	}
}

func toHash(txHash *wire.ShaHash) *common.Hash {
	h := new(common.Hash)
	h.SetBytes(txHash.Bytes())
	return h
}

// UpdateDirBlockInfoMap allows factom processor to update DirBlockInfo
// when a new Directory Block is saved to db
func UpdateDirBlockInfoMap(dirBlockInfo *common.DirBlockInfo) {
	anchorLog.Debug(spew.Sdump(dirBlockInfo))
	dirBlockInfoMap[dirBlockInfo.DBMerkleRoot.String()] = dirBlockInfo
}
