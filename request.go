package collector

import (
    "strings"
    "fmt"
)

const (
    REQUEST_FORMAT     = "request,host=%s,api=%s res=%f,count=%f,interval=%f"
    APP_REQUEST_FORMAT = "app_request,app=%s res=%f,count=%f,interval=%f"
    METRICS_API_PREFIX = "gauge.servo.response"
)

type RequestExtractor struct {
    AbstractExtractor
    kvs []KV
}

func NewRequestExtractor(serviceName, hostName string, timestamp string, metricUrl string, influx *Influx) *RequestExtractor {
    ae := &RequestExtractor{
        kvs: make([]KV, 0),
    }
    ae.timestamp = timestamp
    ae.hostName = hostName
    ae.serviceName = serviceName
    ae.influx = influx
    ae.metricUrl = metricUrl
    return ae
}

func (a *RequestExtractor) extract(key string, value float64) {
    if strings.Index(key, METRICS_API_PREFIX) > -1 {
        api := strings.Replace(key, METRICS_API_PREFIX, "", -1)
        api = strings.Replace(api, ".", "/", -1)
        a.kvs = append(a.kvs, KV{key: api, value: value})

    }
}

func (a *RequestExtractor) process() {
    i := 1
    v := 0.0
    //a.kvs = append(a.kvs, KV{key: "interval", value: float64(a.interval)})

    for _, kv := range a.kvs {
        //MAT = "request,host=%s,api=%s res=%f,count=%f,interval=%f"
        insert := fmt.Sprintf(REQUEST_FORMAT, a.hostName, kv.key, kv.value, float64(1), float64(a.interval))
        i++
        v = v + kv.value
        a.influx.insertDataForApp(a.serviceName, insert, a.timestamp)

        //insertHost := fmt.Sprintf(HOST_RES_FORMAT, a.hostName, kv.value)
        //a.influx.insertDataForApp(a.serviceName, insertHost, a.timestamp)

    }

    avg := v / float64(i)
    //
    //insertApp := fmt.Sprintf(APP_RES_FORMAT, a.hostName, avg)
    //a.influx.insertDataForApp(a.serviceName, insertApp, a.timestamp)

    //app_request,host=%s res=%f,count=%f,interval=%f
    insert := fmt.Sprintf(APP_REQUEST_FORMAT, a.serviceName, avg, float64(i), float64(a.interval))
    a.influx.insertData(a.serviceName, INFLUX_DB_DEFAULT, insert, a.timestamp)

}
