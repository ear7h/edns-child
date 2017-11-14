package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	_passwordFile    = "/var/ear7h/edns/password.txt"
	_masterAdminPort = ":4454"
	_childAdminPort  = ":4455"
	_proxyPort       = ":443"
	_timeout         = (120 / 9) * 10  // timeout in seconds
)

var _hostname string
var _masterHost string
var _password string
var _portMin = 8080
var _portMax = 8090 // exclusive
// subdomain : ip
var _localServices = map[string]string{}
var regLock sync.Mutex

func init() {
	// set the password
	byt, err := ioutil.ReadFile(_passwordFile)
	if err != nil {
		panic(err)
	}
	_password = string(byt)

	// set the hostname
	_hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
	_hostname = strings.ToLower(_hostname)

	// the domain being served
	_masterHost = "ear7h.net"

	// add the admin server as service for the node
	_localServices[_hostname+"."+_masterHost] = "127.0.0.1"+ _childAdminPort
}

// Block is the standard messaging format between child nodes
// and the master node
type Block struct {
	Hostname  string    `json:"hostname"`
	Signature string    `json:"signature"`
	Timestamp time.Time `json:"timestamp"`
	Services  []string  `json:"services"`
	ip        string    // filled in by admin server
}

func signBlock(b *Block) {
	b.Hostname = _hostname
	b.Timestamp = time.Now()

	str := b.Hostname +
		_password +
		b.Timestamp.Format(time.RFC3339Nano) +
		strings.Join(b.Services, "")

	sum := sha512.Sum512([]byte(str))
	b.Signature = base64.StdEncoding.EncodeToString(sum[:])
}

type ClientRequest struct{
	Name string `json:"name"`
	Port uint `json:"port"`
}

type request struct {
	name string
	addr string
}

func register(r request) (resBody []byte, err error) {
	regLock.Lock()

	for k, v := range _localServices {
		if r.addr == v {
			err = fmt.Errorf("Address %s in use by %s", v, k)
			regLock.Unlock()
			return
		}
	}

	_localServices[r.name] = r.addr
	regLock.Unlock()

	b := Block{
		Services: []string{r.name},
	}

	signBlock(&b)

	byt, err := json.Marshal(b)
	if err != nil {
		return
	}

	res, err := http.Post("http://"+_masterHost+_masterAdminPort, "text/json", bytes.NewReader(byt))
	if err != nil {
		return
	}
	resBody, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d response: %s", res.StatusCode, string(resBody))

		regLock.Lock()
		delete(_localServices, r.name)
		regLock.Unlock()
	}

	return

}

func clean() {
	regLock.Lock()
	defer regLock.Unlock()

	fmt.Println("local: ", _localServices)
	for k, v := range _localServices {
		if k == _hostname {
			continue
		}

		conn, err := net.Dial("tcp", v)
		if err != nil {
			delete(_localServices, k)
			continue
		}
		conn.Close()
	}

	fmt.Println("clean: ", _localServices)

}

func post() {
	regLock.Lock()
	services := []string{}

	for k := range _localServices {
		if k == _hostname+"."+_masterHost {
			continue
		}
		services = append(services, k)
	}

	regLock.Unlock()

	b := Block{
		Services: services,
	}

	signBlock(&b)

	fmt.Println("posting: ", b)

	byt, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err)
	}

	_, err = http.Post("http://"+_masterHost+_masterAdminPort, "text/json", bytes.NewReader(byt))
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	go func() {
		for {
			clean()
			post()
			time.Sleep(_timeout * time.Second)
		}
	}()


	go func() {
		panic(serveAdmin())
	}()

	go func() {
		panic(serveRedirect())
	}()

	go func() {
		panic(serveProxy())
	}()

	<- make(chan struct{}, 1)
}
