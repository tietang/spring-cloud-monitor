package collector

import (
    "fmt"
    "strings"
    "bytes"
    "net/http"
    log "github.com/sirupsen/logrus"
    "github.com/serialx/hashring"
)

const (
    INFLUX_DB_DEFAULT              = "default_all"
    INFLUX_WRITE_URL_TEMPLATE      = "%s/write?db=%s"
    INFLUX_QUERY_GET_URL_TEMPLATE  = "%s/query?db=%s"
    INFLUX_QUERY_POST_URL_TEMPLATE = "%s/query"
    CREATE_DATABASE_TEMPLATE       = "CREATE DATABASE %s"
    DROP_DATABASE_TEMPLATE         = "DROP DATABASE %s"
)

type Influx struct {
    urls []string
}

func NewInflux(urls []string) *Influx {
    return &Influx{urls: urls}
}

func (iw *Influx) getNextUrl(key string) string {
    //hash := Runes32([]rune(strings.ToUpper(key)))
    //size := len(iw.urls)
    //if size == 0 {
    //    panic(errs.NilPointError("influx urls is empty"))
    //}
    //index := hash % uint32(size)
    //return iw.urls[index]
    ring := hashring.New(iw.urls)
    server, _ := ring.GetNode(key)
    return server
}

func Runes32(data []rune) uint32 {
    var v uint32 = 2166136261
    for _, c := range data {
        v = (v ^ uint32(c)) * 16777619
    }
    return v
}

func (iw *Influx) createDb(appName, dbName string) {
    cd := fmt.Sprintf(CREATE_DATABASE_TEMPLATE, dbName)
    url := fmt.Sprintf(INFLUX_QUERY_POST_URL_TEMPLATE, iw.getNextUrl(appName))
    log.Info(url, "  ", cd)
    //reader := bytes.NewReader([]byte("q=DROP DATABASE " + toDatabaseName(iw.serviceName)))
    //res, err := http.Post(urls, "application/x-www-form-urlencoded", reader)
    iw.exec(url, cd)
}

func (iw *Influx) exec(url, q string) {

    //reader := bytes.NewReader([]byte("q=DROP DATABASE " + toDatabaseName(iw.serviceName)))
    //res, err := http.Post(urls, "application/x-www-form-urlencoded", reader)

    reader := bytes.NewReader([]byte("q=" + q))
    res, err := http.Post(url, "application/x-www-form-urlencoded", reader)

    //
    if err == nil && res.StatusCode >= 200 && res.StatusCode <= 205 {
        log.Info(q, " Successful!")
    } else {
        log.Error(err, "", res, " failed!")
    }
    //log.Info(res)
}
func (iw *Influx) createDefaultDb() {
    cd := fmt.Sprintf(CREATE_DATABASE_TEMPLATE, INFLUX_DB_DEFAULT)
    for _, url := range iw.urls {
        url = fmt.Sprintf(INFLUX_QUERY_POST_URL_TEMPLATE, url)
        log.Info(url, "  ", cd)

        //reader := bytes.NewReader([]byte("q=DROP DATABASE " + toDatabaseName(iw.serviceName)))
        //res, err := http.Post(urls, "application/x-www-form-urlencoded", reader)
        iw.exec(url, cd)
    }

}
func (iw *Influx) deleteDefaultDb() {
    cd := fmt.Sprintf(DROP_DATABASE_TEMPLATE, INFLUX_DB_DEFAULT)
    for _, url := range iw.urls {
        url = fmt.Sprintf(INFLUX_QUERY_POST_URL_TEMPLATE, url)
        log.Info(url, "  ", cd)
        //reader := bytes.NewReader([]byte("q=DROP DATABASE " + toDatabaseName(iw.serviceName)))
        //res, err := http.Post(urls, "application/x-www-form-urlencoded", reader)
        iw.exec(url, cd)
    }
}
func (iw *Influx) deleteDb(appName, dbName string) {

    cd := fmt.Sprintf(DROP_DATABASE_TEMPLATE, dbName)
    //url := fmt.Sprintf(INFLUX_QUERY_POST_URL_TEMPLATE, iw.getNextUrl(serviceName))
    for _, url := range iw.urls {
        url = fmt.Sprintf(INFLUX_QUERY_POST_URL_TEMPLATE, url)
        log.Info(url, "  ", cd)
        //reader := bytes.NewReader([]byte("q=DROP DATABASE " + toDatabaseName(iw.serviceName)))
        //res, err := http.Post(urls, "application/x-www-form-urlencoded", reader)
        iw.exec(url, cd)
    }
    //
    //reader := bytes.NewReader([]byte("q=" + cd))
    //res, err := http.Post(url, "application/x-www-form-urlencoded", reader)
    //
    ////
    //if err == nil && res.StatusCode >= 200 && res.StatusCode <= 205 {
    //    log.Info(cd, " Successful!")
    //} else {
    //    log.Info(err, " failed!")
    //}
    //log.Info(res)
}
func (iw *Influx) GetDbWriteUrl(appName, dbName string) string {
    db := fmt.Sprintf(INFLUX_WRITE_URL_TEMPLATE, iw.getNextUrl(appName), dbName)
    //urls := "http://172.16.1.248:8086/write?db=demo"
    return db
}

func (iw *Influx) GetDefaultDbUrl(appName string) string {
    return iw.GetDbWriteUrl(appName, INFLUX_DB_DEFAULT)
}

func (iw *Influx) insertData(appName, dbName string, insert string, timestamp string) bool {
    //urls := iw.GetDefaultDbUrl()

    url := iw.GetDbWriteUrl(appName, dbName)
    //metrics,app=x mem=,mem_free=
    insertStmt := insert + " " + timestamp
    reader := bytes.NewReader([]byte(insertStmt))
    //log.Info(urls, " ", insertStmt)
    res, err := http.Post(url, "application/x-www-form-urlencoded", reader)

    if err == nil && res.StatusCode >= 200 && res.StatusCode <= 205 {
        return true
    } else {
        log.Info(url, " ", insertStmt)
        log.Info(dbName, err, " ", res)
    }
    return false
}

func (iw *Influx) insertDataForApp(appName string, insert string, timestamp string) bool {
    dbName := toDatabaseName(appName)
    return iw.insertData(appName, strings.ToUpper(dbName), insert, timestamp)
}

