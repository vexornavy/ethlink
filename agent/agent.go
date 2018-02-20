package agent

import
  (
    "github.com/ethereum/go-ethereum/accounts/keystore"
  )

type Agent struct{
  keystore *keystore.KeyStore
}

//Initialize a new agent
func NewAgent() *Agent {
  ks := keystore.NewKeyStore("/keys/", keystore.StandardScryptN, keystore.StandardScryptP)
  a := Agent{ks}
  return &a
}
