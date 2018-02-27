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
)
var RPC = os.Getenv("RPC_URL")

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

type Agent struct {
  keystore *keystore.KeyStore
  client *client.Client
  tokens map[string]Token
  passwords map[*accounts.Account]Passphrase
}


//Initialize a new agent
func NewAgent() *Agent {
  if RPC == "" {
    RPC = "https://mainnet.infura.io"
  }
  client, _ := client.Dial(RPC)
  tokens := make(map[string]Token)
  passwds := make(map[*accounts.Account]Passphrase)
  ks := keystore.NewKeyStore("keys/", keystore.StandardScryptN, keystore.StandardScryptP)
  a := Agent{ks, client, tokens, passwds}
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
  //save key for 15 minutes
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

  //iterate over every keyfile in the keyfile directory
  names, _ := filepath.Glob("keys/*")
  for _, v := range names {
    //chop off time data from the beginning of filenames, only leaving the address
    addr := v[42:]
    if strings.ToLower(address[2:]) == addr {
      return v, nil
    }
    return "", errors.New("address not found")
  }
  return "", errors.New("unknown error")
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
