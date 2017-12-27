package main

import (
  "html/template"
  "log"
  "net/http"
  "os"
  )

//cache our pages
var tmpls = template.Must(template.ParseFiles("web/index.html", "web/login.html"))

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  http.Handle("/css/", http.FileServer(http.Dir("web")))
  http.Handle("/js/", http.FileServer(http.Dir("web")))
  http.HandleFunc("/login/",loginHandler)
  http.HandleFunc("/", mainHandler)
  http.ListenAndServe(":"+port, nil)
}


func mainHandler(w http.ResponseWriter, r *http.Request) {
  renderTemplate(w, "index.html", nil)
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
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
