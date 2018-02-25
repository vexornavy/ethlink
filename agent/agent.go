package agent

import
  (
    "fmt"
    "time"
    crand "crypto/rand"
    "crypto/md5"
    "errors"
    "path/filepath"
    "strings"

    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/accounts/keystore"
    "github.com/ethereum/go-ethereum/accounts"
    //"github.com/ethereum/go-ethereum/ethclient"
  )

//RPC bits commented out until they're needed
//var RPC = os.Getenv("RPC_URL")

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
  //client *ethclient.Client
  tokens map[string]Token
  passwords map[*accounts.Account]Passphrase
}


//Initialize a new agent
func NewAgent() *Agent {
  //client, _ := ethclient.Dial(RPC)
  tokens := make(map[string]Token)
  passwds := make(map[*accounts.Account]Passphrase)
  ks := keystore.NewKeyStore("keys/", keystore.StandardScryptN, keystore.StandardScryptP)
  a := Agent{ks, tokens, passwds}
  //a := Agent{ks, client, tokens}
  return &a
}

//creates a new token and stores it in the agent
func (a *Agent) CreateToken(account *accounts.Account, permissions string, expiry time.Duration) (token string) {
  b := make([]byte, 32)
  crand.Read(b)
  hash := md5.Sum(b)
  token = fmt.Sprintf("%x", hash)
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

func (a *Agent) KeyfilePath(token string) (path string, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return "", errors.New("token not found")
  }
  if time.Now().After(t.expiry){
    return "", errors.New("token expired")
  }
  address := t.account.Address.Hex()
  names, _ := filepath.Glob("keys/*")
  for _, v := range names {
    //chop off time data from the beginning of filenames
    addr := v[42:]
    if strings.ToLower(address[2:]) == addr {
      return v, nil
    }
  }
  return "", errors.New("address not found")
}
