package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"time"
)

const debug = false
const errorCounterMax = 3
const DeployTo = Boot2Docker

const (
	AWSWithDocker = iota
	Boot2Docker
	NoDocker
)

// Client connection consists of the websocket and the client ip
type Client struct {
	errorCount int
	websocket  *websocket.Conn
	clientIP   string
}

type WordCloud struct {
	activeClients map[string]Client
	IPAddress     string
	Port          string
	errChan       chan error
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (rt *WordCloud) changeIPAddressInFile(filename string, newStr string) error {
	if len(filename) == 0 {
		return fmt.Errorf("Error: invalid len of file")
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	reg := regexp.MustCompile(`[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}:[0-9]{2,5}/sock";`)
	ips := reg.FindAllString(string(b), -1)

	for _, ip := range ips {
		newStr += "/sock\";"
		b = bytes.Replace(b, []byte(ip), []byte(newStr), -1)
	}

	err = ioutil.WriteFile(filename, b, 0644)
	return err
}

// handler for the main page
func HomeHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	webpage, err := ioutil.ReadFile("home.html")

	if err != nil {
		http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
	}

	fmt.Fprint(response, string(webpage))
}

func (rt *WordCloud) broadcastData() {
	var Message = websocket.Message
	var err error
	word_count := 0

	for {
		str := randSeq(10)

		for ip, _ := range rt.activeClients {
			if err = Message.Send(rt.activeClients[ip].websocket, str); err != nil {
				// we could not send the message to a peer
				log.Println("Could not send message to ", ip, err.Error())

				// work-around: https://code.google.com/p/go/issues/detail?id=3117
				var tmp = rt.activeClients[ip]
				tmp.errorCount += 1
				rt.activeClients[ip] = tmp

				if rt.activeClients[ip].errorCount >= errorCounterMax {
					log.Println("Client disconnected:", ip)
					delete(rt.activeClients, ip)
				}
			}
		}

		word_count += 1
		fmt.Println(word_count)

		time.Sleep(100 * time.Millisecond)
	}
}

// reference: https://github.com/Niessy/websocket-golang-chat
// WebSocket server to handle clients
func (rt *WordCloud) WebSocketServer(ws *websocket.Conn) {
	var err error

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	log.Println("New client connected:", client)
	rt.activeClients[client] = Client{0, ws, client}

	// wait for errChan, so the websocket stays open otherwise it'll close
	err = <-rt.errChan
}

func (rt *WordCloud) startHTTPServer() {
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js/"))))
	http.Handle("/", http.HandlerFunc(HomeHandler))
	http.Handle("/sock", websocket.Handler(rt.WebSocketServer))

	err := http.ListenAndServe(":"+rt.Port, nil)
	rt.errChan <- err
}

func NewWordCloud() *WordCloud {
	rt := WordCloud{}
	rt.errChan = make(chan error) // unbuffered channel
	rt.activeClients = make(map[string]Client)
	rand.Seed(time.Now().UTC().UnixNano())
	return &rt
}

func (rt *WordCloud) getExternalIP() string {
	resp, _ := http.Get("http://myexternalip.com/raw")
	defer resp.Body.Close()
	contents, _ := ioutil.ReadAll(resp.Body)
	ip := string(contents)
	return ip[:len(ip)-1]
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	rt := NewWordCloud()

	// change IP Address depending on the env to deploy
	if DeployTo == AWSWithDocker {
		rt.IPAddress = "ebsdockerhellogo-env.elasticbeanstalk.com"
		rt.Port = "80"
	} else if DeployTo == Boot2Docker {
		rt.IPAddress = "192.168.59.103"
		rt.Port = "8080"
	} else if DeployTo == NoDocker {
		rt.IPAddress = rt.getExternalIP()
		rt.Port = "8080"
	}

	var err error
	err = rt.changeIPAddressInFile("home.html", rt.IPAddress+":"+rt.Port)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	go rt.startHTTPServer()
	go rt.broadcastData()

	err = <-rt.errChan

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
