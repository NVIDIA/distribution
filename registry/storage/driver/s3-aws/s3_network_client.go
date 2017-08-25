package s3

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"time"
)

//this client will handle network calls.
//intend for this client to be embedded into s3.Cache
//this client will be initialized when s3.Cache is initialized.

//HTTPClientWrapper is the structure through which s3.Cache obtains relevant credentials
type HTTPClientWrapper struct {
	*http.Client

	//Config contains information and credentials specific to a registry instance
	Config *conf

	//JWT allows the s3_cache to fetch credentials from AuthZ
	JWT string
}

type conf struct {
	ClientID string `yaml:"clientID"`

	Secret string `yaml:"secret"`

	AuthZurl string `yaml:"authZurl"`

	AuthNurl string `yaml:"authNurl"`

	CacheExpiry int64 `yaml:"cacheExpiry"`
}

//getConfig searches for necessary access parameters on disk
func getConfig() (*conf, error) {
	//this will be some other path eventually...
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
	return "", fmt.Errorf("Non-200 response from authN")
}

//GetCredentials returns the AWS credentials for the specific namespace
func (client *HTTPClientWrapper) getCredentials(namespace string, channel chan *Credential) error {
	req, _ := http.NewRequest(
		"GET",
		client.Config.AuthZurl,
		nil,
	)
	req.Header.Set("namespace", namespace)
	req.Header.Set("Authorization", "Bearer "+client.JWT)
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		var credentials Credential
		json.Unmarshal(bodyBytes, &credentials)
		credentials.ValidUntil = time.Now().Add(time.Duration(client.Config.CacheExpiry) * time.Second)

		channel <- &credentials

		return nil
	}
	return fmt.Errorf("Non-200 response from authZ")
	//some form of timeout here would be good
}
