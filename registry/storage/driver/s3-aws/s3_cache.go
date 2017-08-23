package s3

import (
	"fmt"
	"sync"
	//	"time"
	"golang.org/x/sync/syncmap"
	"time"
)

// Cache is a struct that encapsulates all the credential gathering required for s3.go
type Cache struct {
	//MutexMap contains a dynamically allocated map of mutexes for distinct namespaces(keys)
	MutexMap *syncmap.Map

	//CredentialCache maps keys to Credentials
	CredentialCache *syncmap.Map

	//DefaultParameters is a basic set of parameters for s3 driver creation
	DefaultParameters map[string]interface{}

	//Client allows the s3_cache to fetch JWT and credentials as necessary
	Client *HTTPClientWrapper
}

//Credential represents the necessary information to access a unique bucket
type Credential struct {
	Bucket     string
	Access     string
	Secret     string
	ValidUntil time.Time
}

func init() {
	//not really needed. but maybe put other stuff here to clean up.Maybe also some default cache entries?

}

//Initialize returns a new S3Cache instance with a set of default parameters
func Initialize(defaultParameters map[string]interface{}) (*Cache, error) {

	client, _ := NewClient()

	return &Cache{
		MutexMap:          &syncmap.Map{},
		CredentialCache:   &syncmap.Map{},
		DefaultParameters: defaultParameters,
		Client:            client,
	}, nil
}

func (c *Cache) getParams(namespace string) (map[string]interface{}, bool) {

	m := c.returnDefaultParamCopy()

	if params, ok := checkCacheAndUpdate(c.CredentialCache, m, namespace); ok {
		fmt.Print("most thread should hit this case")
		return params, true
	}

	actual, _ := c.MutexMap.LoadOrStore(namespace, &sync.Mutex{})

	if l, ok := actual.(*sync.Mutex); ok {

		l.Lock()

		if params, ok := checkCacheAndUpdate(c.CredentialCache, m, namespace); ok {
			l.Unlock()
			return params, true
		}

		credChan := make(chan *Credential)
		go c.Client.GetCredentials(namespace, credChan)

		credential := <-credChan

		c.CredentialCache.Store(namespace, credential)
		l.Unlock()
	}
	if params, ok := checkCacheAndUpdate(c.CredentialCache, m, namespace); ok {
		return params, true
	}
	//something went wrong.
	return m, false
}

func checkCacheAndUpdate(credentialCache *syncmap.Map, m map[string]interface{}, namespace string) (map[string]interface{}, bool) {
	if v, ok := credentialCache.Load(namespace); ok {
		if x, ok := v.(*Credential); ok {
			if t := time.Now(); t.After(x.ValidUntil) {
				fmt.Println("now invalid, fetch again")
				return m, false
			}
			m["bucket"] = x.Bucket
			m["accesskey"] = x.Access
			m["secretkey"] = x.Secret
			fmt.Print(m)
			//can return here
			return m, true
		}
		//this case bears some thought. it  can hit here if bad credentials are stored!!
		fmt.Println("this will literally not happen")
	}
	return m, false

}

func (c *Cache) returnDefaultParamCopy() map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range c.DefaultParameters {
		newMap[k] = v
	}
	return newMap
}
