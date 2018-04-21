package collector

import (
    "fmt"
    "net/url"
    log "github.com/sirupsen/logrus"
    "strings"
    "time"
    "strconv"
)

const (
    HEALTH_TOTAL = "healths,app=%s up=%d,down=%d"
    HOSTS_STATUS = "hosts,app=%s,host=%s status=%d"
)

func (ca *ServiceCollector) writeDefaultHostsAndAppHealth(timestamp string) {
    upSize := 0
    downSize := 0
    for _, host := range ca.hosts {
        if host == nil {
            continue
        }
        healthUrl := host.HealthCheckUrl
        if strings.TrimSpace(healthUrl) == "" {
            continue
        }
        status := 0
        if ca.HealthOk(healthUrl) {
            upSize++
            status = 1
        } else {
            downSize++
            status = 0
        }
        u, err := url.Parse(healthUrl)
        if err != nil {
            log.Fatal(err)
        }
        host := u.Host
        name := toMeasurementName(ca.serviceName)
        insert := fmt.Sprintf(HOSTS_STATUS, name, host, status)
        ca.influx.insertData(ca.serviceName, INFLUX_DB_DEFAULT, insert, timestamp)
    }
    n := time.Now().UnixNano()
    //插入到default
    insertTotal := fmt.Sprintf(HEALTH_TOTAL, ca.serviceName, upSize, downSize)
    ca.influx.insertData(ca.serviceName, INFLUX_DB_DEFAULT, insertTotal, strconv.Itoa(int(n)))

}
