package collector

import (
    "fmt"
    "strconv"
    log "github.com/sirupsen/logrus"
    "strings"
)

const (
    APP_JVM_FORMAT = "jvm,host=%s "
)

type KV struct {
    key   string
    value float64
}

type Extractor interface {
    extract(key string, value float64)
    process()
}

type AbstractExtractor struct {
    serviceName string
    hostName    string
    timestamp   string
    influx      *Influx
    metricUrl   string
    interval    int
}

type JvmExtractor struct {
    AbstractExtractor
    jvmMetricsGroup map[string][]string
    values          map[string]float64
}

func NewJvmExtractor(serviceName, hostName, timestamp, metricUrl string, interval int, influx *Influx) *JvmExtractor {
    ae := &JvmExtractor{
        values: make(map[string]float64, 0),
    }
    ae.timestamp = timestamp
    ae.hostName = hostName
    ae.serviceName = serviceName
    ae.influx = influx
    ae.metricUrl = metricUrl
    ae.interval = interval
    return ae
}

func (j *JvmExtractor) extract(key string, value float64) {
    //fmt.Println(key, value)
    j.values["interval"] = float64(j.interval)
    for _, mv := range j.jvmMetricsGroup {
        //迭代metrics分组成员信息
        for _, tkey := range mv {
            //metrics key和配置的一样
            //考虑使用正则匹配
            //fmt.Println(tkey," ",k)
            tkey = strings.TrimSpace(tkey)
            isMatch := false
            //用正则来匹配
            //reg := regexp.MustCompile(strings.TrimSpace("gc\\..*\\.count"))
            //fmt.Println(reg.Match([]byte("gc.ps_marksweep.count")))
            if strings.Contains(tkey, "**") {
                ks := strings.Split(tkey, "**")
                size := len(ks)
                if size == 1 {
                    isMatch = strings.Index(key, ks[0]) > -1
                }
                if size > 1 {
                    isMatch = strings.Index(key, ks[0]) > -1 && strings.LastIndex(key, ks[0]) > -1
                }
            }

            if isMatch || key == strings.TrimSpace(tkey) {
                //fmt.Println(key,tkey,value)
                name := toInfluxName(key)
                j.values[name] = value

            }
        }
    }
}

func (j *JvmExtractor) process() {
    insert := fmt.Sprintf(APP_JVM_FORMAT, j.hostName) + " "
    i := 0
    for key, value := range j.values {
        strValue := strconv.FormatFloat(value, 'f', 2, 64)
        if i == 0 {
            insert = insert + key + "=" + strValue
        } else {
            insert = insert + "," + key + "=" + strValue
        }
        i = i + 1
    }
    inserted := j.influx.insertDataForApp(j.serviceName, insert, j.timestamp)
    if !inserted {
        //fmt.Println(insert)
        log.Info("failed: ", j.metricUrl, insert)
    }
    //log.Debug("insert jvm:", insert)
}
