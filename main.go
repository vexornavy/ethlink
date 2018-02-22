package main

import (
  "html/template"
  "log"
  "net/http"
  "os"

  "github.com/vexornavy/ethvault/agent"
  )

//cache our pages
var tmpls = template.Must(template.ParseFiles("web/index.html", "web/login.html", "web/create.html", "web/view.html"))
//figure out if we need to force HTTPS
var ssl = os.Getenv("FORCE_SSL") == "TRUE"
//create agent
var a = agent.NewAgent()
var protocol string

type displayAddr struct {
  Address string
  Key string
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

  //agent := agent.NewAgent()
  http.Handle("/css/", http.FileServer(http.Dir("web")))
  http.Handle("/js/", http.FileServer(http.Dir("web")))
  http.HandleFunc("/login/", loginHandler)
  http.HandleFunc("/create/", createHandler)
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
  renderTemplate(w, "index.html", nil)
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
    token := a.CreateAddress(passphrase)
    addr, key, err := a.GetKey(token)
    if err != nil {
      renderTemplate(w, "index.html", nil)
    }
    p := &displayAddr{addr, key}
    renderTemplate(w, "view.html", p)
    return
  }
  renderTemplate(w, "create.html", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  if ssl {
    redirect := forceSsl(w, r)
    if redirect {
      return
    }
  }
  renderTemplate(w, "login.html", nil)
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}){
  err := tmpls.ExecuteTemplate(w, tmpl, data)
  if err != nil {
    //log error and return it
    log.Println(err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}
