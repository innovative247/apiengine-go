package apiengine

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type apiEngineHandler interface {
	IsAuth() bool
	Get(postfix string) ([]byte, error)
	Post(postfix string, body map[string]string) ([]byte, error)
	loadBaseValues()
	LoadFromConfig(configName string)
}
type apiengineData struct {
	Url           string
	Username      string
	Password      string
	Authorization apiEngineAuth
}
type apiEngineAuth struct {
	Token             string `json:"result"`
	ExpirationSeconds int    `json:"expiration_seconds"`
	ExpiresAt         time.Time
}

var (
	once     sync.Once
	Instance apiEngineHandler = nil
	config   Config
)

const expirationMargin = .9

func (a *apiengineData) Get(postfix string) ([]byte, error) {
	if !a.IsAuth() {
		return nil, nil
	}
	return a.request(http.MethodGet, postfix, nil)
}
func (a *apiengineData) Post(postfix string, body map[string]string) ([]byte, error) {
	if !a.IsAuth() {
		return nil, nil
	}
	return a.request(http.MethodPost, postfix, body)
}
func (a *apiengineData) IsAuth() bool {
	//if there is no token or the time has expired attempt to authorize again
	if (a.Authorization.Token == "") || time.Now().Before(a.Authorization.ExpiresAt) {
		return a.auth()
	}
	return true
}

//Internal functions
func init() {
	once.Do(func() {
		Instance = &apiengineData{}
		Instance.loadBaseValues()
	})
}
func (a *apiengineData) LoadFromConfig(configName string) {
	MergeNewConfig(configName)
	a.loadBaseValues()
}
func (a *apiengineData) loadBaseValues() {
	config = GetConfig()
	//use values if specified in a config file
	baseUrl := config.GetString("apiengine.url")
	baseUsername := config.GetString("apiengine.username")
	basePassword := config.GetString("apiengine.password")
	//use values if specified in startup arguments
	flag.Parse()
	for _, s := range os.Args {
		if strings.HasPrefix(s, "apiengineUrl") {
			//if there is a "next" argument then use it for this value
			baseUrl = getArgValue(s)
		}
		if strings.HasPrefix(s, "apiengineUsername") {
			baseUsername = getArgValue(s)
		}
		if strings.HasPrefix(s, "apienginePassword") {
			basePassword = getArgValue(s)
		}
	}
	a.Url = baseUrl
	a.Username = baseUsername
	a.Password = basePassword
}
func getArgValue(s string) string {
	l := len(s)
	i := strings.Index(s, ":")
	if i == -1 {
		return ""
	}
	return s[(i + 1):l]
}
func (a *apiengineData) auth() bool {
	jsonData := map[string]string{"username": a.Username, "password": a.Password}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(a.Url+"auth", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return false
	} else {
		if response.StatusCode > 299 {
			fmt.Printf("Authorization failed with StatusCode %d\n", response.StatusCode)
			return false
		}
		data, _ := ioutil.ReadAll(response.Body)
		//fmt.Println(string(data))
		a.Authorization, _ = getAuthResult(data)
		//fmt.Println(a)
	}
	return true
}
func getAuthResult(data []byte) (apiEngineAuth, error) {
	var res apiEngineAuth
	//var token1 interface{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		fmt.Println("whoops:", err)
	}
	t := int(float64(res.ExpirationSeconds) * expirationMargin)
	exp := time.Now().Local()
	exp = exp.Add(time.Second * time.Duration((t)))
	res.ExpiresAt = exp
	//fmt.Println(token1.ExpirationSeconds)
	return res, err
}
func (a *apiengineData) request(verb string, postfix string, body map[string]string) ([]byte, error) {
	jsonValue, _ := json.Marshal(body)
	jsonBuff := bytes.NewBuffer(jsonValue)

	fmt.Printf("%s: %s\n", verb, a.Url+postfix)

	//create an http request
	req, err := http.NewRequest(verb, a.Url+postfix, jsonBuff)
	req.Header.Add("User-Token", a.Authorization.Token)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}
	d, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}
	fmt.Println(string(d))
	return d, err
}
