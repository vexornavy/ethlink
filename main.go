package main

import (
  "html/template"
  "log"
  "net/http"
  "os"

  //"github.com/vexornavy/ethvault/agent"
  )

//cache our pages
var tmpls = template.Must(template.ParseFiles("web/index.html", "web/login.html", "web/create.html"))

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
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
  if os.Getenv("FORCE_SSL") == "TRUE" {
    if r.Header.Get("x-forwarded-proto") != "https" {
      sslUrl := "https://" + r.Host + r.RequestURI
      http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
      return true
      }
    }
  return false
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  redirect := forceSsl(w, r)
  if redirect {
    return
  }
  renderTemplate(w, "index.html", nil)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  redirect := forceSsl(w, r)
  if redirect {
    return
  }
  renderTemplate(w, "create.html", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
  //force SSL on heroku
  redirect := forceSsl(w, r)
  if redirect {
    return
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
