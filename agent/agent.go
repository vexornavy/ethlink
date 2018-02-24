package agent

import
  (
    "fmt"
    "time"
    crand "crypto/rand"
    "crypto/md5"
    "errors"

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
  account accounts.Account
  token string
  passphrase string
  permissions string
}

type Agent struct {
  keystore *keystore.KeyStore
  //client *ethclient.Client
  tokens map[string]Token
}


//Initialize a new agent
func NewAgent() *Agent {
  //client, _ := ethclient.Dial(RPC)
  var tokens map[string]Token
  tokens = make(map[string]Token)
  ks := keystore.NewKeyStore("keys/", keystore.StandardScryptN, keystore.StandardScryptP)
  a := Agent{ks, tokens}
  //a := Agent{ks, client, tokens}
  return &a
}

//Create new random key
func (a *Agent) CreateAddress(passphrase string) (token string) {
  account, _ := a.keystore.NewAccount(passphrase)
  //generate random token from the hash
  b := make([]byte, 32)
  crand.Read(b)
  hash := md5.Sum(b)
  token = fmt.Sprintf("%x", hash)
  //store token in the agent
  a.tokens[token] = Token{(time.Now()).Add(time.Minute*15), account, token, passphrase, "view"}
  return
}

func (a *Agent) GetKey(token string) (address string, privateKey string, err error) {
  t, ok := a.tokens[token]
  if !ok {
    return "", "", errors.New("tokenNotPresent")
  }
  if (time.Now()).After(t.expiry) {
    return "", "", errors.New("tokenExpired")
  }
  if t.permissions != "view"{
    return "", "", errors.New("insufficientPermissions")
  }
  passphrase := t.passphrase
  account := t.account
  //load the privatekey of the wallet we just created and convert it to a hex representation
  keyjson, _ := a.keystore.Export(account, passphrase, passphrase)
  key, _ := keystore.DecryptKey(keyjson, passphrase)
  privateKey = fmt.Sprintf("%x", crypto.FromECDSA(key.PrivateKey))
  address = account.Address.Hex()
  return address, privateKey, nil
}
