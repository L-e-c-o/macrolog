package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Conf struct {
	ListenUrl string
	Fullchain string
	Privkey string
	RawLog string
	UsersLog string
	Parameter string
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
	users := strings.Split(buff.String(), "\n")

	return file, users
}

func initLog() *os.File {
	f, err := os.OpenFile(config.RawLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	check(err)
	wrt := io.MultiWriter(os.Stdout, f)
    log.SetOutput(wrt)
	return f
}

func handleRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {  
		param := req.URL.Query()
		if user, ok := param[config.Parameter]; ok {
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
	banner()
	fmt.Println("starting....")
	
	if err := getConfig(); err != nil {
		log.Fatal(err)
	}

	fLog := initLog()	
	defer fLog.Close()

	fUsers, users := load()
	defer fUsers.Close()	

	go checkUser(users, fUsers)

	http.HandleFunc("/", handleRequest)

	err := http.ListenAndServeTLS(config.ListenUrl, config.Fullchain, config.Privkey, nil)
	check(err)
}
