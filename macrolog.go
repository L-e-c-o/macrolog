package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"fmt"
)

var channel = make(chan string)

type conf struct {
	ListenUrl string
	Fullchain string
	Privkey string
	RawLog string
	UsersLog string
	Parameter string
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getConfig() conf {
	c, err := ioutil.ReadFile("config.json")
	check(err)
	var config conf
	err = json.Unmarshal(c, &config)
	check(err)
	

	return config
}

func contains(slice []string, pattern string) bool {
	for _, v := range slice {
		if v == pattern {
			return true
		}
	}
	return false
}

func load(usersLog string) (*os.File, []string) {
	file, err := os.OpenFile(usersLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	check(err)
	buff := new(bytes.Buffer)
	buff.ReadFrom(file)
	check(err)
	users := strings.Split(buff.String(), "\n")

	return file, users
}

func initLog(rawLog string) *os.File {
	f, err := os.OpenFile(rawLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	check(err)
	wrt := io.MultiWriter(os.Stdout, f)
    log.SetOutput(wrt)
	return f
}

func handleRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {  
		param := req.URL.Query()
		if user, ok := param["id"]; ok {
			log.Println(user[0], req.RemoteAddr, "\""+req.UserAgent()+"\"")
			channel <-user[0]
			w.WriteHeader(http.StatusOK)
			return
		} else {
			goto EXIT
		}
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

func main() {
	fmt.Println("starting....")
	
	config := getConfig()

	fLog := initLog(config.RawLog)	
	defer fLog.Close()

	fUsers, users := load(config.UsersLog)
	defer fUsers.Close()	

	go checkUser(users, fUsers)

	http.HandleFunc("/", handleRequest)

	err := http.ListenAndServeTLS(config.ListenUrl, config.Fullchain, config.Privkey, nil)
	check(err)
}
