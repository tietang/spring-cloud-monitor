package lock

import (
    "github.com/garyburd/redigo/redis"
    "time"
    "github.com/kataras/go-errors"
    "github.com/tietang/props"
)

type RedisConfig struct {
    Namespace   string
    Address     string
    Password    string
    Db          int
    TTL         int
    Retries     int
    MaxIdle     int
    MaxActive   int
    IdleTimeout time.Duration
}

func (r *RedisConfig) Init() {
    if r.TTL <= 0 {
        r.TTL = 3
    }

    if r.Db < 0 || r.Db > 15 {
        r.Db = 0
    }
    if r.MaxIdle <= 0 {
        r.MaxIdle = 2
    }
    if r.MaxActive <= 0 {
        r.MaxActive = 5
    }
    if r.IdleTimeout <= 0 {
        r.IdleTimeout = 1 * time.Second
    }

}

func NewRedisConfig(conf props.ConfigSource) *RedisConfig {
    config := &RedisConfig{
        Address:     conf.GetDefault("redis.address", "127.0.0.1:6379"),
        Password:    conf.GetDefault("redis.password", ""),
        Db:          conf.GetIntDefault("redis.db", 0),
        MaxActive:   conf.GetIntDefault("redis.MaxActive", 3),
        MaxIdle:     conf.GetIntDefault("redis.MaxIdle", 2),
        IdleTimeout: conf.GetDurationDefault("redis.IdleTimeout", 2*time.Second),
        TTL:         conf.GetIntDefault("redis.ttl", 0),
        Namespace:   conf.GetDefault("redis.namespace", ""),
        Retries:     conf.GetIntDefault("redis.retries", 3),
    }
    return config
}

type RedisLock struct {
    RedisConfig
    pool *redis.Pool
}

func NewRedisLock(config *RedisConfig) *RedisLock {
    config.Init()
    r := &RedisLock{}
    r.RedisConfig = *config

    r.pool = &redis.Pool{
        MaxIdle:     r.MaxIdle,
        MaxActive:   r.MaxActive,
        IdleTimeout: r.IdleTimeout,

        // Other pool configuration not shown in this example.
        Dial: func() (redis.Conn, error) {
            c, err := redis.Dial("tcp", r.Address)
            if err != nil {
                return nil, err
            }
            if r.Password != "" {
                if _, err := c.Do("AUTH", r.Password); err != nil {
                    c.Close()
                    return nil, err
                }
            }
            if _, err := c.Do("SELECT", r.Db); err != nil {
                c.Close()
                return nil, err
            }
            return c, nil
        },
        // Other pool configuration not shown in this example.
        TestOnBorrow: func(c redis.Conn, t time.Time) error {
            if time.Since(t) < time.Minute {
                return nil
            }
            _, err := c.Do("PING")
            return err
        },
    }
    return r
}

func (r *RedisLock) LockDefault(key string) (bool, error) {
    return r.Lock(key, r.TTL)
}

func (r *RedisLock) Lock(key string, timeout int) (bool, error) {
    f := func(conn redis.Conn) (bool, error) {
        k := r.toLockKey(key)
        rp, err := conn.Do("setnx", k, time.Now().Unix())
        conn.Do("EXPIRE", k, timeout)
        if err != nil {
            return false, err
        }
        ok, err := redis.Bool(rp, err)
        if !ok {
            return false, err
        }

        return ok, err
    }
    return r.exec(f)

}

func (r *RedisLock) Unlock(key string) {
    r.exec(func(conn redis.Conn) (bool, error) {
        k := r.toLockKey(key)
        rp, err := conn.Do("del", k)
        if err != nil {
            return false, err
        }
        ok, err := redis.Bool(rp, err)
        if !ok {
            return false, err
        }

        return ok, err
    })
}

func (r *RedisLock) exec(f func(redis.Conn) (bool, error)) (bool, error) {
    c := r.pool.Get()
    ////redis.Dial("tcp", redisAddr)
    //if err != nil {
    //    // handle error
    //    fmt.Println(err)
    //}

    defer c.Close()

    for i := 0; i < r.Retries; i++ {
        ok, e := f(c)
        if ok {
            return ok, e
        }
    }
    return false, errors.New("can't get lock, timeout or has been locked.")
}

func (r *RedisLock) toLockKey(key string) string {
    if r.Namespace == "" {
        return "LOCK:" + key
    } else {
        return "LOCK:" + r.Namespace + ":" + key
    }

}
