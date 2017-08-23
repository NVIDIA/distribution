package s3

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

//this client will handle network calls.
//intend for this client to be embedded into s3.Cache
//this client will be initialized when s3.Cache is initialized.

//HTTPClientWrapper is the structure through which s3.Cache obtains relevant credentials
type HTTPClientWrapper struct {
	*http.Client

	Config *conf

	//JWT allows the s3_cache to fetch credentials from AuthZ
	//does this need to be a pointer?
	JWT string
}

type conf struct {
	ClientID string `yaml:"clientID"`

	Secret string `yaml:"secret"`

	AuthZurl string `yaml:"authZurl"`

	AuthNurl string `yaml:"authNurl"`

	CacheExpiry int64 `yaml:"cacheExpiry"`
}

var lock sync.Mutex

//getConfig searches for necessary access parameters on disk
func getConfig() (*conf, error) {
	bytes, err := ioutil.ReadFile("/home/howard/testConfigs/config.yaml")

	if err != nil {
		return nil, err
	}

	var c conf
	err = yaml.Unmarshal(bytes, &c)

	if err != nil {
		return nil, err
	}

	return &c, nil

}

//NewClient returns a new instance of the HTTPClientWrapper
func NewClient() (*HTTPClientWrapper, error) {
	c, err := getConfig()

	if err != nil {
		return nil, err
	}

	client := &HTTPClientWrapper{
		Client: &http.Client{},
		Config: c,
	}

	jwt, err := client.getJWT()
	if err != nil {
		return nil, err
	}
	client.JWT = jwt

	return client, nil
}

//getJWT returns the identifying JWT for the registry instance.
func (client *HTTPClientWrapper) getJWT() (string, error) {
	//that should be an error instead of _
	//that can be handled later when i introduce proper error handling
	req, err := http.NewRequest(
		"GET",
		client.Config.AuthNurl,
		nil,
	)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(client.Config.ClientID, client.Config.Secret)
	resp, _ := client.Do(req)

	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return string(bodyBytes), nil
	}
	return "", errors.New("An unexpected error has occurred, a JWT cannot be fetched")
}

//GetCredentials returns the AWS credentials for the specific namespace
func (client *HTTPClientWrapper) GetCredentials(namespace string, channel chan *Credential) {
	req, _ := http.NewRequest(
		"GET",
		client.Config.AuthZurl,
		nil,
	)
	fmt.Print("get credential cal")
	req.Header.Set("namespace", namespace)
	req.Header.Set("Authorization", "Bearer"+client.JWT)
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		var credentials Credential
		json.Unmarshal(bodyBytes, &credentials)
		credentials.ValidUntil = time.Now().Add(time.Duration(client.Config.CacheExpiry) * time.Second)

		channel <- &credentials
	}
	//some form of timeout here would be good
}
