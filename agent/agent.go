package agent

import
  (
    "os"
    "fmt"
    "time"
    crand "crypto/rand"
    "errors"
    "path/filepath"
    "strings"
    "encoding/hex"
    "math/big"
    "context"
    "log"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/accounts/keystore"
    "github.com/ethereum/go-ethereum/accounts"
    client "github.com/ethereum/go-ethereum/ethclient"
  )

//long scale
const (
  Thousand = 1000
  Million = Thousand * Thousand
  Milliard = Million * Thousand
  Billion = Million * Million
  Trillion = Million * Billion
)
var RPC = os.Getenv("RPC_URL")
var test = os.Getenv("ETHVAULT_ENV")
var chainID *big.Int

//TODO : periodically delete expired tokens
type Token struct {
  expiry time.Time
  account *accounts.Account
  token string
  permissions string
}

type Passphrase struct {
  expiry time.Time
  passphrase string
}

type Transaction struct {
  expiry time.Time
  transaction *types.Transaction
}

type Agent struct {
  keystore *keystore.KeyStore
  client *client.Client
  tokens map[string]Token
  passwords map[*accounts.Account]Passphrase
  txQueue map[string]Transaction
}


//Initialize a new agent
func NewAgent() *Agent {
  var network string
  if test == "test" || test == "TEST" {
    chainID = big.NewInt(3)
    if RPC == "" {
      RPC = "https://ropsten.infura.io"
    }
    network = "ropsten test network"
  } else {
    chainID = big.NewInt(1)
    if RPC == "" {
      RPC = "https://mainnet.infura.io"
    }
    network = "mainnet"
  }
  client, _ := client.Dial(RPC)
  tokens := make(map[string]Token)
  passwds := make(map[*accounts.Account]Passphrase)
  queue := make(map[string]Transaction)
  ks := keystore.NewKeyStore("keys/", keystore.StandardScryptN, keystore.StandardScryptP)
  a := Agent{ks, client, tokens, passwds, queue}
  log.Println("agent initialized on " + network)
  time.Sleep(time.Millisecond*100)
  return &a
}

//creates a new token and stores it in the agent
func (a *Agent) CreateToken(account *accounts.Account, permissions string, expiry time.Duration) (token string) {
  b := make([]byte, 32)
  crand.Read(b)
  token = fmt.Sprintf("%x", b)
  //store token in the agent
  a.tokens[token] = Token{(time.Now()).Add(expiry), account, token, permissions}
  return token
}

//Create new random key
func (a *Agent) CreateAddress(passphrase string) (account *accounts.Account) {
  acc, _ := a.keystore.NewAccount(passphrase)
  //store the account for an hour
  a.passwords[&acc] = Passphrase{time.Now().Add(time.Hour), passphrase}
  return &acc
}

func (a *Agent) GetKey(account *accounts.Account) (privateKey string, err error) {
  passphrase, ok := a.passwords[account]
  if !ok {
    return "", errors.New("address not found")
  }
  if time.Now().After(passphrase.expiry) {
    return "", errors.New("address expired")
  }
  secret := passphrase.passphrase
  //load the privatekey of the wallet we just created and convert it to a hex representation
  keyjson, _ := a.keystore.Export(*account, secret, secret)
  key, _ := keystore.DecryptKey(keyjson, secret)
  privateKey = fmt.Sprintf("%x", crypto.FromECDSA(key.PrivateKey))
  return privateKey, nil
}

//return the path to the keyfile a given token gives access to
func (a *Agent) KeyfilePath(token string) (path string, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return "", errors.New("token not found")
  }
  if time.Now().After(t.expiry){
    return "", errors.New("token expired")
  }
  if t.permissions != "download" {
    return "", errors.New("token invalid")
  }
  address := t.account.Address.Hex()
  log.Println("looking for", address)

  //iterate over every keyfile in the keyfile directory
  names, _ := filepath.Glob("keys/*")
  for _, v := range names {
    //chop off time data from the beginning of filenames, only leaving the address
    addr := v[42:]
    log.Println(addr)
    if strings.ToLower(address[2:]) == addr {
      return v, nil
    }
  }
  return "", errors.New("address not found")
}

//import a JSON keyfile, given as a byte array to keystore
func (a *Agent) ImportKeyfile(keyfile []byte, passphrase string) (account *accounts.Account, err error) {
  acc, err := a.keystore.Import(keyfile, passphrase, passphrase)
  //store the account for an hour
  a.passwords[&acc] = Passphrase{time.Now().Add(time.Hour), passphrase}
  return &acc, nil
}

//import a hex string representation of a private key to keystore
func (a *Agent) ImportKey(privatekey string) (account *accounts.Account, err error) {
  key_d, err := hex.DecodeString(privatekey)
  if err != nil {
    return nil, err
  }
  key, err := crypto.ToECDSA(key_d)
  if err != nil {
    return nil, err
  }
  //if this account is in the keystore already, return the account we already have
  addr := crypto.PubkeyToAddress(key.PublicKey)
  if a.keystore.HasAddress(addr) {
    acc := accounts.Account{Address: addr}
    acc, err := a.keystore.Find(acc)
    if err != nil {
      return nil, err
    }
    a.passwords[&acc] = Passphrase{time.Now().Add(time.Hour), ""}
    return &acc, nil
  }
  acc, err := a.keystore.ImportECDSA(key, "")
  if err != nil {
    return nil, err
  }
  //store the account for an hour
  a.passwords[&acc] = Passphrase{time.Now().Add(time.Hour), ""}
  return &acc, nil
}

//returns account's balance in ETH to the nearest Szano/microether
func (a *Agent) GetBalance(acc *accounts.Account) (balance float64, err error) {
  bal, err := a.client.PendingBalanceAt(context.TODO(), acc.Address)
  //convert balance from wei to Szabo (10^-6 eth)
  //integer division, may not be precise
  bal = bal.Div(bal, big.NewInt(Billion))
  //convert balance from Szabo to Ether
  balance = float64(bal.Int64()) / Million
  return balance, err
}

//returns account's nonce
func (a *Agent) GetNonce(acc *accounts.Account) (nonce uint64, err error) {
  nonce, err = a.client.PendingNonceAt(context.TODO(), acc.Address)
  return nonce, err
}

//returns estimated gas price as Gwei
func (a *Agent) EstimateGas() (gasprice float64, err error) {
  gas, err := a.client.SuggestGasPrice(context.TODO())
  gasprice = float64(gas.Int64()) / Milliard
  return
}

func (a *Agent) NewTx(nonce uint64, to common.Address, amount float64, gasLimit uint64, gasPrice float64, token string) (tx *types.Transaction, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return nil, errors.New("token not found")
  }
  if time.Now().After(t.expiry){
    return nil, errors.New("token expired")
  }
  if t.permissions != "send" {
    return nil, errors.New("token invalid")
  }

  //no data - empty array
  var d []byte

  //convert amount from ether (float64) to wei(*big.Int)
  x := big.NewFloat(amount)
  tril := big.NewFloat(float64(Trillion))
  x = x.Mul(x, tril)
  amt, _ := x.Int(nil)

  //convert amount from gwei(float64) to wei(*big.Int)
  x = big.NewFloat(gasPrice)
  mrd := big.NewFloat(float64(Milliard))
  x = x.Mul(x, mrd)
  gprice, _ := x.Int(nil)

  tx = types.NewTransaction(nonce, to, amt, gasLimit, gprice, d)
  return tx, nil
}

func (a *Agent) QueueTx(tx *types.Transaction, token string) (txToken string, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return "", errors.New("token not found")
  }
  if time.Now().After(t.expiry) {
    return "", errors.New("token expired")
  }
  if t.permissions != "send" {
    return "", errors.New("token invalid")
  }
  acc := t.account
  passphrase := a.passwords[acc].passphrase
  tx, err = a.keystore.SignTxWithPassphrase(*acc, passphrase, tx, chainID)
  if err != nil {
    return "", err
  }
  b := make([]byte, 32)
  crand.Read(b)
  txToken = fmt.Sprintf("%x", b)
  a.txQueue[txToken] = Transaction{time.Now().Add(time.Minute*20), tx}
  return txToken, nil
}

func (a *Agent) GetAccount(token string) (acc *accounts.Account, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return nil, errors.New("token not found")
  }
  if time.Now().After(t.expiry) {
    return nil, errors.New("token expired")
  }
  acc = t.account
  return acc, nil
}

func (a *Agent) clearExpired() {
  //remove every expired token
  for k, v := range a.tokens {
    if time.Now().After(v.expiry) {
      delete(a.tokens, k)
    }
  }
  //same but passwords
  for k, v := range a.passwords {
    if time.Now().After(v.expiry) {
      //remove expired key from keystore
      a.keystore.Delete(*k, v.passphrase)
      delete(a.passwords, k)
    }
  }
  //same but transactions
  for k, v := range a.txQueue {
    if time.Now().After(v.expiry) {
      delete(a.txQueue, k)
    }
  }
  return
}

func (a *Agent) gcLoop() {
  //trigger garbage collection routine every 15 minutes automatically
  for {
    log.Println("clearing all expired data...")
    a.clearExpired()
    log.Println("done")
    time.Sleep(time.Minute*15)
  }
  return
}
