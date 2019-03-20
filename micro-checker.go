package collector

import (
    "github.com/tietang/go-eureka-client/eureka"
    "github.com/tietang/props/kvs"
    "time"
    "strings"
    "github.com/rcrowley/go-metrics"
    "net/http"
    "net"
    "sync"
    "github.com/go-redis/redis"
    log "github.com/sirupsen/logrus"
    "encoding/json"
)

const (
    CONF_EUREKA_URLS_TEMPLATE   = "eureka.urls"
    CONF_EUREKA_INTERVAL_SECOND = "eureka.interval.second"
    CONF_CHECK_INTERVAL_SECOND  = "checker.interval.second"
    //CONF_HEALTH_4XX_SLEEP       = "checker.health.4xx.sleep"
    CONF_REDIS_ENABLED  = "redis.enabled"
    CONF_REDIS_ADDR     = "redis.addr"
    CONF_REDIS_PASSWORD = "redis.password"
    CONF_REDIS_DB       = "redis.db"
    DEFAULT_REDIS_DB    = 0

    //DEFAULT_HEALTH_4XX_SLEEP = 10 * time.Minute
    DATE_FORMAT = "2006-01-02.15:04:05.999999"
    //
    REDIS_KEY_HEALTH_SERVICES = "health:services"
)

type Service struct {
    Name   string
    Status bool
    //当前健康度，共10分，满10分表示ok，0表示全不ok
    //1,5,15分钟内的健康度
    Health1m  int
    Health5m  int
    Health15m int
    Instances []*Instance
    //MicroHealthChecker *MicroHealthChecker `json:"-"`
    discovery *eureka.Discovery  `json:"-"`
    ticker    *time.Ticker       `json:"-"`
    conf      kvs.ConfigSource `json:"-"`
    //config
    ServiceId string
    //
    lastCheckTime string

    health4xx *sync.Map
    redis     *redis.Client
}

type Instance struct {
    Id     string `json:"-"`
    Name   string
    Status bool   `json:"-"`
    //当前健康度，共10分，满10分表示ok，0表示全不ok
    //1,5,15分钟内的健康度
    Health1m       int                  `json:"1m"`
    Health5m       int                  `json:"5m"`
    Health15m      int                  `json:"15m"`
    HealthCheckUrl string
    meterUp        metrics.Meter        `json:"-"`
    meterDown      metrics.Meter        `json:"-"`
    InstanceInfo   *eureka.InstanceInfo `json:"-"`
}

type HealthChecker struct {
    discovery *eureka.Discovery
    conf      kvs.ConfigSource
    //config
    eurekaUrls     []string
    eurekaInterval time.Duration

    Services map[string]*Service
    //MicroHealthCheckers map[string]*MicroHealthChecker
    redis *redis.Client
}

func NewHealthChecker(conf kvs.ConfigSource) *HealthChecker {
    var redisClient *redis.Client
    if conf.GetBoolDefault(CONF_REDIS_ENABLED, false) {
        redisClient = redis.NewClient(&redis.Options{
            Addr:     conf.GetDefault(CONF_REDIS_ADDR, "127.0.0.1:6379"),
            Password: conf.GetDefault(CONF_REDIS_PASSWORD, ""),                  // no password set
            DB:       conf.GetIntDefault(CONF_REDIS_PASSWORD, DEFAULT_REDIS_DB), // use default DB
        })
        log.Info(redisClient.Info())
        pong, err := redisClient.Ping().Result()
        log.Info(pong, err)
    }

    h := &HealthChecker{
        conf:     conf,
        Services: make(map[string]*Service),
        redis:    redisClient,
        //MicroHealthCheckers: make(map[string]*MicroHealthChecker),
    }

    return h
}

func (h *HealthChecker) Start() {
    timeout := h.conf.GetDurationDefault("", 10*time.Second)
    http.DefaultTransport = &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
            DualStack: true,
        }).DialContext,
        MaxIdleConns:          200,
        MaxIdleConnsPerHost:   3,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        Dial: func(network, addr string) (net.Conn, error) {
            //deadline := time.Now().Add(10 * time.Second)
            c, err := net.DialTimeout(network, addr, timeout) //设置建立连接超时
            if err != nil {
                return nil, err
            }
            //c.SetDeadline(deadline) //不建议，设置发送接收数据超时
            return c, nil
        },
    }

    //h.startEurekaClient()
}

func (h *HealthChecker) update(apps *eureka.Applications) {

    if apps == nil || len(apps.Applications) == 0 {
        return
    }
    for _, app := range apps.Applications {
        appName := strings.ToUpper(app.Name)
        c, ok := h.Services[appName]
        isNewService := false
        if !ok {

            c = &Service{
                discovery: h.discovery,
                conf:      h.conf,
                ServiceId: appName,
                Instances: make([]*Instance, 0),
                Name:      app.Name,
                health4xx: new(sync.Map),
                redis:     h.redis,
            }
            isNewService = true
            if h.conf.GetBoolDefault(CONF_REDIS_ENABLED, false) {
                val, err := h.redis.SAdd(REDIS_KEY_HEALTH_SERVICES, appName).Result()
                log.Info(REDIS_KEY_HEALTH_SERVICES, " ", appName, " ", val)
                if err != nil {
                    log.Info(err, val)
                }
            }
        } else {
            isNewService = false
        }

        for _, ins := range app.Instances {
            ok, idx := c.exits(&ins)
            if ok {
                c.Instances[idx].InstanceInfo = &ins
            } else {
                //port := ins.Port.Port
                //if ins.SecurePort.Enabled {
                //    port = ins.SecurePort.Port
                //}
                //strPort := strconv.Itoa(port)
                //id := strings.Join([]string{strings.ToUpper(appName), ins.HostName, strPort}, ":")
                id, strPort := c.id(&ins)
                instance := &Instance{
                    Id:             id,
                    Name:           strings.Join([]string{ins.IpAddr, strPort}, ":"),
                    InstanceInfo:   &ins,
                    HealthCheckUrl: ins.HealthCheckUrl,
                    Status:         true,
                    meterUp:        metrics.GetOrRegisterMeter("UP:"+id, metrics.DefaultRegistry),
                    meterDown:      metrics.GetOrRegisterMeter("DOWN:"+id, metrics.DefaultRegistry),
                }
                c.Instances = append(c.Instances, instance)
            }

        }
        h.Services[appName] = c
        if isNewService {
            h.Services[appName].Start()
        }
    }
}

func (h *HealthChecker) GetServices() map[string]*Service {
    if h.conf.GetBoolDefault(CONF_REDIS_ENABLED, false) {
        services := make(map[string]*Service)
        values, e := h.redis.SMembers(REDIS_KEY_HEALTH_SERVICES).Result()
        if e != nil {
            log.Error(e)
            return h.Services
        }

        for _, serviceId := range values {
            services[serviceId] = &Service{
                Name:      serviceId,
                ServiceId: serviceId,
                Health1m:  -1,
                Health5m:  -1,
                Health15m: -1,
                Instances: make([]*Instance, 0),
            }
            instanceValues, e := h.redis.HGetAll(serviceId).Result()
            if e != nil {
                log.Error(e)
                continue
            }
            for name, value := range instanceValues {
                instance := &Instance{}
                e := json.Unmarshal([]byte(value), instance)
                if e != nil {
                    log.Error(e)
                    continue
                }
                instance.Id = name

                services[serviceId].Instances = append(services[serviceId].Instances, instance)
            }

        }

        return services
    } else {
        return h.Services
    }

}

func (h *HealthChecker) ResetServices(services map[string]*Service) map[string]*Service {
    for _, value := range services {
        //fmt.Print(key, ":")
        health1m := [2]int{}
        health5m := [2]int{}
        health15m := [2]int{}
        for _, ins := range value.Instances {
            ins.Health1m = reset(ins.Health1m)
            ins.Health5m = reset(ins.Health5m)
            ins.Health15m = reset(ins.Health15m)
            health1m[0] += ins.Health1m
            health1m[1] ++
            health5m[0] += ins.Health5m
            health5m[1] ++
            health15m[0] += ins.Health15m
            health15m[1] ++

        }
        value.Health1m = health1m[0] / health1m[1]
        value.Health5m = health5m[0] / health5m[1]
        value.Health15m = health15m[0] / health15m[1]

        value.Health1m = reset(value.Health1m)
        value.Health5m = reset(value.Health5m)
        value.Health15m = reset(value.Health15m)
    }
    return services
}

func reset(v int) int {
    if v < 0 {
        return -1
    }
    return v
}
