package lock

import (
    consulapi "github.com/hashicorp/consul/api"
    "github.com/hashicorp/go-multierror"
    "github.com/tietang/props/kvs"
    "strconv"
)

type ConsulConfig struct {
    Namespace string
    Address   string
    TTL       int
    Retries   int
}

func (r *ConsulConfig) Init() {
    if r.TTL < 10 {
        r.TTL = 10
    }

    if r.Retries <= 0 {
        r.Retries = 0
    }
}

func NewConsulConfig(conf kvs.ConfigSource) *ConsulConfig {
    config := &ConsulConfig{
        Address:   conf.GetDefault("consul.address", "127.0.0.1:6379"),
        TTL:       conf.GetIntDefault("consul.ttl", 0),
        Namespace: conf.GetDefault("consul.namespace", ""),
        Retries:   conf.GetIntDefault("consul.retries", 3),
    }
    return config
}

type ConsulLock struct {
    Client *consulapi.Client
    config *ConsulConfig
}

func NewConsulLock(cc *ConsulConfig) (*ConsulLock, error) {

    config := &consulapi.Config{
        Address: cc.Address,
        Scheme:  "http",
    }

    client, err := consulapi.NewClient(config)
    if err != nil {
        return &ConsulLock{}, err
    }

    return &ConsulLock{
        Client: client,
    }, nil
}

func (c *ConsulLock) CreateSession(sessionName string) (string, error) {
    session := c.Client.Session()
    sessionID, _, err := session.Create(&consulapi.SessionEntry{
        Behavior: consulapi.SessionBehaviorDelete,
        TTL:      strconv.Itoa(c.config.TTL) + "s",
        Name:     sessionName,
    }, nil)
    session.Renew(sessionID, nil)
    return sessionID, err
}

func (c *ConsulLock) destroySession(sessionID string) error {
    session := c.Client.Session()
    _, err := session.Destroy(sessionID, nil)
    return err
}

func (c *ConsulLock) AcquireLock(key, sessionID string) (bool, error) {
    //value := strconv.Itoa(int(time.Now().Unix()))

    kv := c.Client.KV()
    kvpair := &consulapi.KVPair{
        Key:     key,
        Session: sessionID,
        Value:   []byte(sessionID),
    }
    ok, _, err := kv.Acquire(kvpair, nil)
    //kv.Put(kvpair, nil)

    //log.Fatal(ok, m, err)
    return ok, err
}

func (c *ConsulLock) ReleaseLock(key, sessionID string) error {
    var result error

    kv := c.Client.KV()

    _, _, err := kv.Release(&consulapi.KVPair{
        Key:     key,
        Session: sessionID,
        Value:   []byte(""),
    }, nil)
    if err != nil {
        result = multierror.Append(result, err)
    }

    err = c.destroySession(sessionID)
    if err != nil {
        result = multierror.Append(result, err)
    }

    return result
}
