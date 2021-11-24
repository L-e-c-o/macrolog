package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

type Conf struct {
	ListenUrl string
	Fullchain string
	Privkey string
	RawLog string
	UsersLog string
	Parameter string
	Method string
	Route string
}

var channel = make(chan string)
var config Conf

func banner() {
	fmt.Print(`


                                                      $$\                     
                                                      $$ |                    
$$$$$$\$$$$\   $$$$$$\   $$$$$$$\  $$$$$$\   $$$$$$\  $$ | $$$$$$\   $$$$$$\  
$$  _$$  _$$\  \____$$\ $$  _____|$$  __$$\ $$  __$$\ $$ |$$  __$$\ $$  __$$\ 
$$ / $$ / $$ | $$$$$$$ |$$ /      $$ |  \__|$$ /  $$ |$$ |$$ /  $$ |$$ /  $$ |
$$ | $$ | $$ |$$  __$$ |$$ |      $$ |      $$ |  $$ |$$ |$$ |  $$ |$$ |  $$ |
$$ | $$ | $$ |\$$$$$$$ |\$$$$$$$\ $$ |      \$$$$$$  |$$ |\$$$$$$  |\$$$$$$$ |
\__| \__| \__| \_______| \_______|\__|       \______/ \__| \______/  \____$$ |
                                                                    $$\   $$ |
                                                                    \$$$$$$  |
                                                                     \______/ 
                                                                              
	 	      made with  ♥  by leco & atsika
			   
`)

}

func check(err error) {
	if err != nil {
		log.Println(err)
	}
}

func getConfig() error {
	c, err := ioutil.ReadFile("config.json")
	check(err)
	err = json.Unmarshal(c, &config)
	check(err)

	return err
}

func contains(slice []string, pattern string) bool {
	for _, v := range slice {
		if v == pattern {
			return true
		}
	}
	return false
}

func load() (*os.File, []string) {
	file, err := os.OpenFile(config.UsersLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	check(err)
	
	buff := new(bytes.Buffer)
	buff.ReadFrom(file)
	check(err)
	
	var users []string
	if len(buff.Bytes()) != 0 { 
		users = strings.Split(buff.String(), "\n")
	}

	return file, users
}

func initLog() {
	f, err := os.OpenFile(config.RawLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	check(err)
	wrt := io.MultiWriter(os.Stdout, f)
    log.SetOutput(wrt)

}

func handleRequest(w http.ResponseWriter, req *http.Request) {
	var user string
	if config.Method == http.MethodPost { 
		err := req.ParseForm()
		check(err)
		user = strings.TrimSuffix(req.PostForm.Get(config.Parameter), "\r\n")
	} else if config.Method == http.MethodGet {
		param := req.URL.Query()
		user = param[config.Parameter][0]	
	}
	if user != "" {
		log.Println(user, req.RemoteAddr, "\""+req.UserAgent()+"\"")
		channel <-user
		w.WriteHeader(http.StatusOK)
		return
	} else {
		goto EXIT
	}
	EXIT:
		http.NotFound(w, req)
}

func checkUser(users []string, fUsers *os.File) {
	for {
		user := <-channel

		if !contains(users, user){
			users = append(users, user)
			if _, err := fUsers.WriteString(user+"\n"); err != nil {
				panic(err)
			}	
		} 
	}
}

func configTls(mux *mux.Router) *http.Server {
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	srv := &http.Server{
		Addr:         config.ListenUrl,
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	return srv
}

func main() {
	banner()
	fmt.Println("starting....")
	
	if err := getConfig(); err != nil {
		log.Fatal(err)
	}

	initLog()

	fUsers, users := load()

	go checkUser(users, fUsers)
	
	r := mux.NewRouter()
	r.HandleFunc(config.Route, handleRequest).Methods(config.Method)

	srv := configTls(r)

	log.Fatal(srv.ListenAndServeTLS(config.Fullchain, config.Privkey))
}