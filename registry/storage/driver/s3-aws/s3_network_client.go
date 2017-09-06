package s3

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	JWT "github.com/dgrijalva/jwt-go"
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

	//Config contains information and credentials specific to a registry instance
	Config *conf

	//JWT allows the s3_cache to fetch credentials from AuthZ
	JWT string

	//JWTExpiry represents the expiry time of the current JWT in storage
	JWTExpiry time.Time

	//JWTLock blocks all requests while waiting for JWT from AuthN
	JWTLock *sync.Mutex
}

type conf struct {
	ClientID string `yaml:"clientID"`

	Secret string `yaml:"secret"`

	AuthZurl string `yaml:"authZurl"`

	AuthNurl string `yaml:"authNurl"`

	CacheExpiry int `yaml:"cacheExpiry"`

	PublicKeyurl string `yaml:"publicKeyurl"`

	JWTValidityOffset int64 `yaml:"jwtValidityOffset"`
}

type token struct {
	Token string `json:"token"`
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
		Client:    &http.Client{},
		Config:    c,
		JWTExpiry: time.Time{},
		JWTLock:   &sync.Mutex{},
	}

	return client, nil
}

//getJWT returns the identifying JWT for the registry instance.
func (client *HTTPClientWrapper) getJWT() (string, time.Time, error) {

	req, err := http.NewRequest(
		"GET",
		client.Config.AuthNurl,
		nil,
	)
	if err != nil {
		return "", time.Time{}, err
	}
	req.SetBasicAuth(client.Config.ClientID, client.Config.Secret)
	resp, _ := client.Do(req)

	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		var t token
		json.Unmarshal(bodyBytes, &t)

		jwtExpiry, err := client.getJWTExpiry(t.Token)

		if err != nil {
			return "", time.Time{}, nil
		}

		return t.Token, jwtExpiry, nil
	}
	return "", time.Time{}, fmt.Errorf("Non-200 response from authN, received %d instead", resp.StatusCode)
}

func (client *HTTPClientWrapper) getJWTExpiry(jwt string) (time.Time, error) {
	token, err := JWT.ParseWithClaims(jwt, &JWT.StandardClaims{}, func(token *JWT.Token) (interface{}, error) {

		pubkey, err := client.getPublicKey()
		if err != nil {
			return nil, err
		}
		return pubkey, err
	})

	if err != nil {
		return time.Time{}, err
	}

	if claims, ok := token.Claims.(*JWT.StandardClaims); ok && token.Valid {

		return time.Now().Add(
			time.Duration(
				claims.ExpiresAt-claims.IssuedAt-client.Config.JWTValidityOffset) * time.Second), nil
	}

	if !token.Valid {
		return time.Time{}, fmt.Errorf("Token invalid")
	}

	return time.Time{}, fmt.Errorf("Claims malformed")

}

func (client *HTTPClientWrapper) getPublicKey() (*rsa.PublicKey, error) {
	req, _ := http.NewRequest(
		"GET",
		client.Config.PublicKeyurl,
		nil,
	)
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		pubkey, err := JWT.ParseRSAPublicKeyFromPEM(bodyBytes)
		if err != nil {
			return nil, err
		}
		return pubkey, err
	}
	return nil, fmt.Errorf("Unable to get publickey, received %d instead", resp.StatusCode)
}

//GetCredentials returns the AWS credentials for the specific namespace
func (client *HTTPClientWrapper) getCredentials(namespace string) (*Credential, error) {
	//First, check if the jwt is valid
	if t := time.Now(); t.After(client.JWTExpiry) {

		client.JWTLock.Lock()
		//Second layer check to prevent redundant calls
		if t := time.Now(); t.After(client.JWTExpiry) {
			fmt.Print("fetching new jwt")
			jwt, jwtExpiry, err := client.getJWT()
			if err != nil {
				client.JWTLock.Unlock()
				return nil, err
			}
			client.JWT = jwt
			client.JWTExpiry = jwtExpiry
		}

		client.JWTLock.Unlock()
	}

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

		return &credentials, nil
	}
	return nil, fmt.Errorf("Non-200 response from authZ, received %d instead", resp.StatusCode)
	//some form of timeout here would be good
}
