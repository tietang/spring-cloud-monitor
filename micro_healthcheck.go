package collector

import (
    "time"
    "github.com/tietang/go-eureka-client/eureka"
    "github.com/rcrowley/go-metrics"
    "net/http"
    "strings"
    "strconv"
    "net"
    "encoding/json"
    log "github.com/sirupsen/logrus"
)

var appCounter metrics.Counter
var hostCounter metrics.Counter

func init() {
    appCounter = metrics.NewCounter()
    metrics.Register("serviceCounter", appCounter)
    hostCounter = metrics.NewCounter()
    metrics.Register("hostTotalCounter", hostCounter)
    http.DefaultClient.Transport = &http.Transport{
        DialContext: (&net.Dialer{
            Timeout:   10 * time.Second,
            KeepAlive: 600 * time.Second,
            DualStack: true,
        }).DialContext,
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   5,
        IdleConnTimeout:       900 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }
    http.DefaultClient.Timeout = 10 * time.Second
}

//
//type Service struct {
//    discovery *eureka.Discovery
//    ticker    *time.Ticker
//    conf      props.ConfigSource
//    //config
//    ServiceId string
//    Instances []*Instance
//    //
//    lastCheckTime string
//}

func (h *Service) Start() {

    h.ticker = time.NewTicker(h.conf.GetDurationDefault(CONF_CHECK_INTERVAL_SECOND, 3*time.Second))
    go func() {
        for t := range h.ticker.C {
            for _, instance := range h.Instances {
                if &instance == nil {
                    continue
                }
                healthCheckUrl := instance.HealthCheckUrl
                go h.HealthOk(instance, healthCheckUrl)
                metrics.GetOrRegisterCounter(h.ServiceId, nil).Inc(1)
                hostCounter.Inc(1)

            }
            h.lastCheckTime = t.Format(DATE_FORMAT)
        }
    }()
}

func (h *Service) HealthOk(instance *Instance, url string) bool {
    //监测4xx的`/health` sleep时间是否已过
    v, ok := h.health4xx.Load(url)
    if ok && v != nil {
        health4xxSleep := h.conf.GetDurationDefault(CONF_HEALTH_4XX_SLEEP, DEFAULT_HEALTH_4XX_SLEEP)
        t := v.(time.Time)
        n := time.Now()
        n.Add(-health4xxSleep)
        if t.After(time.Now()) {
            h.health4xx.Store(url, nil)
        } else {
            return false
        }
    }
    //请求health endpoint
    res, err := http.Get(url)
    if err == nil {
        defer res.Body.Close()
    }
    if err == nil && res.StatusCode == 200 {
        instance.Status = true
        instance.meterUp.Mark(1)
    } else {
        if res != nil && res.StatusCode >= 400 && res.StatusCode < 500 {
            h.health4xx.Store(url, time.Now())
        } else {
            if res == nil {
                log.Error(url, " ", err)
            } else {
                log.Error(url, " ", res.StatusCode, " ", err)
            }
            instance.meterDown.Mark(1)
            instance.Status = false
        }
    }
    //实时计算当前
    instance.Health1m = int(0.5 + 10*instance.meterUp.Rate1()/(instance.meterUp.Snapshot().Rate1()+instance.meterDown.Snapshot().Rate1()))
    instance.Health5m = int(0.5 + 10*instance.meterUp.Rate5()/(instance.meterUp.Snapshot().Rate5()+instance.meterDown.Snapshot().Rate5()))
    instance.Health15m = int(0.5 + 10*instance.meterUp.Rate15()/(instance.meterUp.Snapshot().Rate15()+instance.meterDown.Snapshot().Rate15()))
    if h.conf.GetBoolDefault(CONF_REDIS_ENABLED, false) {
        serviceId := strings.ToUpper(h.ServiceId)
        data, err := json.Marshal(instance)
        val, err := h.redis.HSet(serviceId, instance.Name, string(data)).Result()
        //glog.Info(string(data))
        if err != nil {
            log.Error(err, val)
        }
    }
    return instance.Status
}

func (h *Service) exits(ins *eureka.InstanceInfo) (bool, int) {
    for i, instance := range h.Instances {
        id, _ := h.id(ins)
        if instance.Id == id {
            return true, i
        }
    }
    return false, -1
}

func (h *Service) id(ins *eureka.InstanceInfo) (string, string) {
    port := ins.Port.Port
    if ins.SecurePort.Enabled {
        port = ins.SecurePort.Port
    }
    strPort := strconv.Itoa(port)
    id := strings.Join([]string{strings.ToUpper(ins.AppName), ins.HostName, strPort}, ":")
    return id, strPort
}
