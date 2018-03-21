package main

import (
  "html/template"
  "log"
  "net/http"
  "os"
  "time"
  "io/ioutil"
  "errors"
  "strings"
  conv "strconv"

  "github.com/vexornavy/ethvault/agent"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts"
  )

//cache our pages
var templates map[string]*template.Template
//figure out if we need to force HTTPS
var ssl = os.Getenv("FORCE_SSL") == "TRUE"
var test = os.Getenv("ETHVAULT_ENV") == "test"
//create agent
var a = agent.NewAgent()
var protocol string

//Structs for giving data to templates
type displayAddr struct {
  Address string
  Key string
  Token string
  Path string
}
type send struct {
  Balance float64
  GasPrice float64
  Nonce uint64
  Token string
  Path string
}
type confirm struct {
  Amount string
  To string
  From string
  Token string
  Path string
}
type sendPrefilled struct {
  Error string
  Balance float64
  Address string
  Amount float64
  GasPrice float64
  Nonce uint64
  Token string
  Path string
}
type errorSplash struct {
  Error string
  Path string
}
type Path struct {
  Path string
}
type showTx struct {
  Txid string
  Path string
}

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  if ssl {
    protocol = "https://"
  } else {
    protocol = "http://"
  }

  //initialize templates
  templates = make(map[string]*template.Template)
  tlist := []string{"send", "index", "login", "create", "view", "error", "confirm", "sent"}
  templates = make(map[string]*template.Template)
  for _, name := range tlist {
    t := template.Must(template.New("layout").ParseFiles("web/layout.html", "web/" + name + ".html"))
    templates[name] = t
  }
  //if running on test environment
  if test {
    t := template.Must(template.New("layout").ParseFiles("web/layout.html","web/sent_test.html"))
    templates["sent"] = t
    t = template.Must(template.New("layout").ParseFiles("web/layout.html","web/test.html"))
    templates["index"] = t
  }
  //load handler functions
  log.Println("Ethvault running on port :"+port)
  http.Handle("/css/", http.FileServer(http.Dir("web")))
  http.Handle("/js/", http.FileServer(http.Dir("web")))
  http.HandleFunc("/access/", authHandler)
  http.HandleFunc("/create/", createHandler)
  http.HandleFunc("/confirm/", confirmHandler)
  http.HandleFunc("/download/", downloadHandler)
  http.HandleFunc("/login/", loginHandler)
  http.HandleFunc("/sent/", senderHandler)
  http.HandleFunc("/", mainHandler)
  http.ListenAndServe(":"+port, nil)
}


//force SSL helper when running on heroku
//code source: github.com/jonahgeorge/force-ssl-heroku
func forceSsl(w http.ResponseWriter, r *http.Request) bool {
  if ssl {
    if r.Header.Get("x-forwarded-proto") != "https" {
      sslUrl := "https://" + r.Host + r.URL.Path
      http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
      return true
      }
    }
  return false
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }
  //redirect the user to root if their request is weird
  if r.URL.Path != "/" {
    rootUrl := protocol + r.Host + "/"
    http.Redirect(w, r, rootUrl, http.StatusTemporaryRedirect)
    return
  }
  renderTemplate(w, r, "index", nil)
}

func confirmHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }

  if r.Method == "POST" {
    token := r.FormValue("token")
    rc := r.FormValue("address")
    if !common.IsHexAddress(rc){
      //TODO: txfail form
      handleErr(w, r, errors.New("not an address"))
      return
    }
    amount := r.FormValue("amount")
    gasprice := r.FormValue("gasprice")
    gaslimit := r.FormValue("gaslimit")
    nonce := r.FormValue("nonce")
    if amount == "" || gasprice == "" || gaslimit == "" || nonce == "" {
      handleErr(w, r, errors.New("Some of the fields seem to be empty"))
      return
    }
    nc, _ := conv.ParseUint(nonce, 10, 64)
    gl, _ := conv.ParseUint(gaslimit, 10, 64)
    //convert floats to use a full stop instead of a comma
    gasprice = strings.Replace(gasprice, ",", ".", 1)
    amount = strings.Replace(amount, ",", ".", 1)
    amt, _ := conv.ParseFloat(amount, 64)
    gp, _ := conv.ParseFloat(gasprice, 64)
    if amt == 0 || gp == 0 || gl == 0 {
      handleErr(w, r, errors.New("amount, gas price or gas limit can't be zero"))
      return
    }
    recipient := common.HexToAddress(rc)
    tx, err := a.NewTx(nc, recipient, amt, gl, gp, token)
    if err != nil {
      handleErr(w, r, err)
      return
    }
    txToken, err := a.QueueTx(tx, token)
    sender, _ := a.GetAccount(token)
    from := sender.Address.Hex()
    if err != nil {
      handleErr(w, r, err)
      return
    }
    d := confirm{amount, rc, from, txToken, "../"}
    renderTemplate(w, r, "confirm", d)
    return
  }

  redirect(w, r)
  return
}

//redirect to root
func redirect(w http.ResponseWriter, r *http.Request) {
  rootUrl := protocol + r.Host + "/"
  http.Redirect(w, r, rootUrl, http.StatusTemporaryRedirect)
  return
}

func authHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }

  if r.Method == "POST" {
    var err error
    var account *accounts.Account
    log.Println(r.URL.RawQuery)
    if r.URL.RawQuery == "file" {
      err := r.ParseMultipartForm(16384)
      if err != nil {
        handleErr(w, r, err)
        return
      }
      keyfile, header, err := r.FormFile("keyfile")
      passphrase := r.FormValue("passphrase")
      if header.Size > 2048 {
        handleErr(w, r, errors.New("keyfile looks invalid"))
        return
      }
      var keyjson []byte
      keyjson, err = ioutil.ReadAll(keyfile)
      account, err = a.ImportKeyfile(keyjson, passphrase)
      if err != nil {
        handleErr(w, r, err)
        return
      }
    } else {
      privatekey := r.FormValue("privatekey")
      account, err = a.ImportKey(privatekey)
      if err != nil {
        handleErr(w, r, err)
        return
      }
    }
    balance, _ := a.GetBalance(account)
    nonce, _ := a.GetNonce(account)
    token := a.CreateToken(account, "send", time.Minute*20)
    gasprice, _ := a.EstimateGas()
    p := send{balance, gasprice, nonce, token, "../"}
    renderTemplate(w, r, "send", p)
    return
  }
  redirect(w, r)
  return
}

func createHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }
  if r.Method == "POST" {
    passphrase := r.FormValue("passphrase")
    account := a.CreateAddress(passphrase)
    key, err := a.GetKey(account)
    token := a.CreateToken(account, "download", time.Minute*30)
    if err != nil {
      handleErr(w, r, err)
      return
    }
    addr := account.Address.Hex()
    p := displayAddr{addr, key, token, "../"}
    renderTemplate(w, r, "view", p)
    return
  }
  renderTemplate(w, r, "create", nil)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }

  if r.Method == "POST" {
    token := r.FormValue("token")
    path, err := a.KeyfilePath(token)
    if err != nil {
      handleErr(w, r, err)
      return
    }
    w.Header().Set("Content-Disposition", "attachment; filename=" + path[42:] + ".json")
    http.ServeFile(w, r, path)
    return
  }
  rootUrl := protocol + r.Host + "/"
  http.Redirect(w, r, rootUrl, http.StatusTemporaryRedirect)
  return
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }
  renderTemplate(w, r, "login", nil)
}

func senderHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }

  if r.Method == "POST" {
    token := r.FormValue("token")
    hash, err := a.SendTx(token)
    if err != nil {
      handleErr(w, r, err)
      return
    }
    d := showTx{hash, "../"}
    renderTemplate(w, r, "sent", d)
    return
  }
  //redirect to root if this is not a post Request
  redirect(w, r)
  return
}

func handleErr(w http.ResponseWriter, r *http.Request, err error) {
   log.Println(err.Error())
   e := errorSplash{err.Error(), "../"}
   renderTemplate(w, r, "error", e)
 }

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}){
  //if we're not at root, set path to "../", else empty string.
  if data == nil {
    path := ""
    if r.URL.Path != "/"{
      path = "../"
    }
    data = Path{path}
  }
  t := templates[tmpl]
  err := t.Execute(w, data)
  if err != nil {
    //log error and return it
    log.Println(err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}
