package collector

import (
    "github.com/tietang/props/kvs"
    "time"
    "math"
    "strconv"
    "net/url"
    "io/ioutil"
    "net/http"
    "encoding/json"
    "github.com/rcrowley/go-metrics"
    "github.com/tietang/go-eureka-client/eureka"
    log "github.com/sirupsen/logrus"
    "strings"
    "reflect"
    "github.com/tietang/spring-cloud-monitor/lock"
    "fmt"
    "sync"
)

const (
    CONF_METRICS_JVM_GROUP    = "metrics.jvm.group"
    CONF_METRICS_JVM_TEMPLATE = "metrics.jvm.%s"

    SERVICE_VALUE            = "doing"
    CONF_HEALTH_4XX_SLEEP    = "checker.health.4xx.sleep"
    DEFAULT_HEALTH_4XX_SLEEP = 10 * time.Minute

    HOSTS = "hosts,app=%s,host=%s status=%d"
)

type MetricsGroup struct {
    prefix string
    name   string
    keys   []string
}

type Host struct {
    Id             string
    HomePageUrl    string
    HealthCheckUrl string
    status         int
}

type ServiceCollector struct {
    serviceName string
    conf        kvs.ConfigSource
    collector   *Collector

    influx *Influx

    hosts                []*Host
    jvmMetricsGroup      map[string][]string
    lastCollectTime      string
    lastCollectTimeKey   string
    lastCollectHostCount int
    //
    redisConfig *lock.RedisConfig
    redisLock   *lock.RedisLock
    //consulLock *lock.ConsulLock
    health4xx *sync.Map
}

func NewServiceCollector(serviceName string, collector *Collector) *ServiceCollector {
    s := &ServiceCollector{
        hosts:       make([]*Host, 0),
        conf:        collector.conf,
        serviceName: serviceName,
        influx:      collector.influx,
        collector:   collector,
        health4xx:   new(sync.Map),
    }
    s.redisLock = lock.NewRedisLock(collector.redisConfig)
    return s
}

func (ca *ServiceCollector) AddOrUpdateHost(app *eureka.Application) {
    for _, ins := range app.Instances {
        h := &Host{
            HomePageUrl:    ins.HomePageUrl,
            HealthCheckUrl: ins.HealthCheckUrl,
            Id:             ins.HomePageUrl,
        }
        ca.AddHost(h)
    }

}
func (ca *ServiceCollector) AddHost(host *Host) {
    isExists := false
    for i, h := range ca.hosts {
        if h.Id == host.Id {
            ca.hosts[i] = host
            isExists = true
        }
    }
    if !isExists {
        ca.hosts = append(ca.hosts, host)
        log.Debug("append host:", host)
    }
}

func (ca *ServiceCollector) Start() {
    dbName := toDatabaseName(ca.serviceName)

    //if ca.conf.GetBoolDefault("influx.auto.delete", false) {
    //    ca.influx.deleteDefaultDb()
    //}
    //
    ca.influx.createDb(ca.serviceName, dbName)
    interval := ca.collector.interval
    ca.collector.addJob(ca.serviceName, func() {
        now := time.Now().Unix()
        //整形
        x := math.Floor(float64(now) / interval.Seconds())
        n := (time.Duration(x*interval.Seconds()) * time.Second).Nanoseconds()

        timestamp := strconv.Itoa(int(n))

        ok := ca.exec(int(n), func() {
            //ips, _ := utils.GetExternalIPs()
            //p := ca.conf.GetDefault("http.server.port", "8888")
            //log.Info("lock: ", ips[0], ":", p, ca.serviceName)

            for _, host := range ca.hosts {
                if host == nil {
                    continue
                }

                metricsUrl := host.HomePageUrl + "metrics"
                if strings.LastIndex(host.HomePageUrl, "/") == -1 {
                    metricsUrl = host.HomePageUrl + "/metrics"
                }

                u, err := url.Parse(metricsUrl)
                if err != nil {
                    log.Fatal(err)
                }
                //
                e := make([]Extractor, 0)
                e = append(e, NewRequestExtractor(ca.serviceName, u.Host, timestamp, metricsUrl, ca.influx))
                jvm := NewJvmExtractor(ca.serviceName, u.Host, timestamp, metricsUrl, int(interval/time.Second), ca.influx)
                jvm.jvmMetricsGroup = ca.jvmMetricsGroup
                e = append(e, jvm)
                go ca.writeMetrics(host, e, metricsUrl, timestamp)
                //go ca.writeHosts(host, timestamp)
                metrics.GetOrRegisterCounter(ca.serviceName, nil).Inc(1)
            }

            ca.writeDefaultHostsAndAppHealth(timestamp)
        })
        if ok {
            counter := metrics.GetOrRegisterCounter(ca.serviceName, metrics.DefaultRegistry)
            counter.Inc(1)
        }
        //metrics.Log(metrics.DefaultRegistry, 30*time.Second, ca)
    })

}
func (ca *ServiceCollector) Printf(format string, v ...interface{}) () {
    fmt.Printf(format, v)

}

func (ca *ServiceCollector) writeHosts(host *Host, timestamp string) {
    //
    dbName := toDatabaseName(ca.serviceName)
    //status := ca.exsitsHost(host)
    hostsInsert := fmt.Sprintf(HOSTS, dbName, host.Id, 1)
    ca.influx.insertData(ca.serviceName, INFLUX_DB_DEFAULT, hostsInsert, timestamp)

}

func (ca *ServiceCollector) exsitsHost(host *Host) int {
    if len(ca.hosts) == 0 {
        return 0
    }
    for _, h1 := range ca.hosts {
        if h1.Id == host.Id {
            return 1
        }
    }
    return 0

}

func (ca *ServiceCollector) exec(seed int, run func()) bool {
    key := ca.serviceName
    ok, err := ca.redisLock.LockDefault(key)
    //log.Error(ok, err)
    if !ok || err != nil {
        return false
    }
    if ok {
        run()
        return true
    }

    //ca.redisLock.Unlock(key)
    return true
}

//从远程读取meitrics信息，根据本地配置的
func (iw *ServiceCollector) writeMetrics(host *Host, extractors []Extractor, urlStr, timestamp string) {
    //监测4xx的`/health` sleep时间是否已过
    v, ok := iw.health4xx.Load(urlStr)
    if ok && v != nil {
        health4xxSleep := iw.conf.GetDurationDefault(CONF_HEALTH_4XX_SLEEP, DEFAULT_HEALTH_4XX_SLEEP)
        t := v.(time.Time)
        n := time.Now()
        n.Add(-health4xxSleep)
        if t.After(time.Now()) {
            iw.health4xx.Store(urlStr, nil)
        } else {
            return
        }
    }
    res, err := http.Get(urlStr)
    if res != nil && res.StatusCode >= 400 && res.StatusCode < 500 {
        iw.health4xx.Store(urlStr, time.Now())
        return
    }
    if err == nil && res.StatusCode == 200 {
        body, err := ioutil.ReadAll(res.Body)
        if err != nil {
            return
        }
        d := make(map[string]interface{})
        err = json.Unmarshal(body, &d)

        status := iw.HealthOk(host.HealthCheckUrl)
        if status {
            d["health_status"] = 1
        } else {
            d["health_status"] = 0
        }

        if err == nil {
            //迭代远程metrics
            for k, v := range d {
                for _, extractor := range extractors {
                    value := reflect.ValueOf(v)
                    typ := value.Type()

                    if typ.Kind() == reflect.Float64 {
                        //fmt.Println(typ, typ.Name(), typ.Kind())

                        extractor.extract(k, v.(float64))
                    }

                }
            }
        } else {
            log.Error(err)
        }

        for _, extractor := range extractors {
            extractor.process()
        }

    } else {
        log.Error(urlStr, err, res)
    }
}

func (iw *ServiceCollector) HealthOk(url string) bool {
    res, err := http.Get(url)
    if err == nil && res.StatusCode == 200 {
        return true
    }
    return false
}
