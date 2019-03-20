package collector

import (
	"encoding/json"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"github.com/rcrowley/go-metrics"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"github.com/tietang/go-eureka-client/eureka"
	"github.com/tietang/go-utils"
	"github.com/tietang/props/ini"
	"github.com/tietang/props/kvs"
	"github.com/tietang/spring-cloud-monitor/lock"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"
)

const (
	CONF_METRICS_GROUP_PREFEX     = "metrics."
	KEY_EUREKA_URL                = "eureka.urls"
	KEY_EUREKA_DISCOVERY_INTERVAL = "eureka.interval"
	DISCOVERY_INTERVAL_DEFAULT    = 10 * time.Second

	CONF_INFLUX_URLS = "influx.urls"

	CONF_COLLECT_INTERVAL      = "collector.interval"
	DEFAULT_COLLECT_INTERVAL   = 10 * time.Second
	CONF_COLLECT_CRON_TEMPLATE = "%d/%d * * * * ?"
	LB_FACTOR                  = 10
)

type Collector struct {
	conf      kvs.ConfigSource
	urls      []string
	discovery *eureka.Discovery
	checker   *HealthChecker
	//collectors      []*ServiceCollector
	collectors      map[string]*ServiceCollector
	influx          *Influx
	jvmMetricsGroup map[string][]string
	interval        time.Duration
	cron            *cron.Cron
	redisConfig     *lock.RedisConfig
}

func NewCollector(conf kvs.ConfigSource) *Collector {
	e := &Collector{
		conf:            conf,
		jvmMetricsGroup: make(map[string][]string, 0),
		collectors:      make(map[string]*ServiceCollector),
	}
	e.redisConfig = lock.NewRedisConfig(conf)
	e.cron = cron.New()
	e.interval = conf.GetDurationDefault(CONF_COLLECT_INTERVAL, DEFAULT_COLLECT_INTERVAL)
	e.checker = NewHealthChecker(conf)
	e.cron.Start()
	return e
}

func NewCollectorByFile(name string) *Collector {
	conf := ini.NewIniFileConfigSource(name)
	e := NewCollector(conf)
	return e
}

func (c *Collector) addJob(serviceName string, f func()) {
	//取服务名称的hash值，并依次递增分为LB_FACTOR个区间
	i := utils.Hash([]byte(serviceName), LB_FACTOR)
	index := i % LB_FACTOR
	cronExp := fmt.Sprintf(CONF_COLLECT_CRON_TEMPLATE, index, int(c.interval.Seconds()))
	log.Println(cronExp)
	c.cron.AddFunc(cronExp, f)
}

func (c *Collector) collect(apps *eureka.Applications) {
	if apps == nil {
		log.Warn("apps is nil")
		return
	}
	for _, app := range apps.Applications {
		c.AddOrUpdateCollector(&app)
	}
}

func (c *Collector) AddOrUpdateCollector(app *eureka.Application) {
	if iw, ok := c.collectors[app.Name]; ok {
		iw.AddOrUpdateHost(app)
	} else {
		iw := NewServiceCollector(app.Name, c)
		iw.jvmMetricsGroup = c.jvmMetricsGroup
		iw.AddOrUpdateHost(app)
		//c.collectors = append(c.collectors, iw)
		c.collectors[iw.serviceName] = iw
		iw.Start()
	}
}

func (e *Collector) StartEureka(callbacks ...func(*eureka.Applications)) {

	urls := e.conf.Strings(KEY_EUREKA_URL)
	log.Info("erueka urls: ", urls)
	//eurekaUrls := strings.Split(urls, ",|, | , ")
	e.urls = urls
	discovery := eureka.NewDiscovery(e.urls)
	e.discovery = discovery
	apps := e.discovery.GetApps()
	//e.collect(apps)
	for _, cb := range callbacks {
		cb(apps)
	}
	e.discovery.ScheduleAtFixedRate(e.conf.GetDurationDefault(KEY_EUREKA_DISCOVERY_INTERVAL, DISCOVERY_INTERVAL_DEFAULT))
	//e.discovery.AddCallback(e.collect)
	for _, cb := range callbacks {
		//cb(apps)
		e.discovery.AddCallback(cb)
	}
	//
}
func (e *Collector) Start() {

	//
	e.GetJvmMetrics()
	//
	//iu, err := e.conf.Get(CONF_INFLUX_URLS)
	//if err != nil {
	//    log.Error(err.Error())
	//}
	//ius := strings.Split(iu, ",")
	ius := e.conf.Strings(CONF_INFLUX_URLS)
	log.Info("influxdb urls: ", ius)
	influx := NewInflux(ius)

	if e.conf.GetBoolDefault("influx.auto.delete", false) {
		influx.deleteDefaultDb()
	}

	influx.createDefaultDb()
	e.influx = influx
	e.StartEureka(e.collect, e.checker.update)
	//
	//urls := e.conf.GetDefault(KEY_EUREKA_URL, "http://127.0.0.1:8761/eureka")
	//
	//eurekaUrls := strings.Split(urls, ",|, | , ")
	//e.urls = eurekaUrls
	//discovery := eureka.NewDiscovery(e.urls)
	//e.discovery = discovery
	//e.collect(e.discovery.GetApps())
	//e.discovery.ScheduleAtFixedRate(e.conf.GetDurationDefault(KEY_EUREKA_DISCOVERY_INTERVAL, DISCOVERY_INTERVAL_DEFAULT))
	//e.discovery.AddCallback(e.collect)
	//
	port := e.conf.GetDefault("http.server.port", "8088")

	checker := e.checker
	checker.Start()
	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())
	log.Info(os.Getwd())

	tmpl := iris.HTML("./public/views", ".html")

	app.Favicon("./public/favicon.ico")
	tmpl.Reload(true) // reload templates on each request (development mode)
	app.RegisterView(tmpl)
	app.StaticWeb("/assets", "./assets")
	app.Get("/", func(ctx context.Context) {
		ctx.Gzip(true)
		services := checker.ResetServices(checker.GetServices())
		ctx.ViewData("services", services)
		ctx.View("health.html")
	})
	app.Get("/metrics", func(ctx context.Context) {
		m := make(map[string]interface{})
		m["metrics"] = metrics.DefaultRegistry
		for _, iw := range e.collectors {
			iwm := make(map[string]interface{})
			iwm[ "lastCollectTime"] = iw.lastCollectTime
			iwm[ "lastCollectHostCount"] = iw.lastCollectHostCount
			iwm[ "lastCollectTimeKey"] = iw.lastCollectTimeKey
			m[iw.serviceName] = iwm
		}

		data, err := json.Marshal(m)

		if err != nil {
			ctx.Write([]byte(err.Error()))
			return
		}
		ctx.Write(data)
	})
	//port := conf.GetDefault("http.server.port", "8080")
	app.Run(iris.Addr(":"+port), iris.WithCharset("UTF-8"), iris.WithoutServerError(iris.ErrServerClosed))
	log.Info("service started. for server port：", port)
	//log.Fatal(http.ListenAndServe(":"+port, nil))
	//log.Println()
	//wg := &sync.WaitGroup{}
	//wg.Add(1)
	//wg.Wait()
}

func (iw *Collector) GetJvmMetrics() {

	for _, k := range iw.conf.Keys() {
		v, e := iw.conf.Get(k)
		if e != nil {
			continue
		}

		if strings.Index(k, CONF_METRICS_GROUP_PREFEX) > -1 {
			ks := strings.Split(k, ".")

			if len(ks) >= 2 {
				vs := strings.Split(v, ",")
				iw.jvmMetricsGroup[k] = vs
			}
		}

	}
}
