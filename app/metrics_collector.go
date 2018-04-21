package main

import (
    log "github.com/sirupsen/logrus"
    "github.com/mattn/go-colorable"
    "github.com/tietang/spring-cloud-monitor"
    "github.com/tietang/go-utils"
    "time"
    "net/http"
    "net"
)

const (
    DATE_FORMAT = "2006-01-02.15:04:05.999999"
)

func init() {

    //formatter := &log.TextFormatter{}
    //formatter.ForceColors = true
    //formatter.DisableColors = false
    //formatter.FullTimestamp = true
    //formatter.TimestampFormat = "2006-01-02.15:04:05.999999"
    //

    //formatter := &prefixed.TextFormatter{}
    formatter := &utils.TextFormatter{}
    formatter.ForceColors = true
    formatter.DisableColors = false
    formatter.FullTimestamp = true
    formatter.ForceFormatting = true
    formatter.EnableLogLine = true
    formatter.EnableLogFuncName = true
    //formatter.EnableLogFuncName = false
    formatter.SetColorScheme(&utils.ColorScheme{
        InfoLevelStyle:  "green",
        WarnLevelStyle:  "yellow",
        ErrorLevelStyle: "red",
        FatalLevelStyle: "red",
        PanicLevelStyle: "red",
        DebugLevelStyle: "blue",
        PrefixStyle:     "cyan+b",
        TimestampStyle:  "black+h",
    })
    formatter.TimestampFormat = "2006-01-02.15:04:05.000000"
    log.SetFormatter(formatter)
    log.SetOutput(colorable.NewColorableStdout())
    //log.SetOutput(os.Stdout) propfile
    //log.SetLevel(log.WarnLevel)

    //log.SetOutput(os.Stdout) propfile
    log.SetLevel(log.DebugLevel)

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
func main() {
    c := collector.NewCollectorByFile("conf.ini")
    //c := collector.NewCollectorByFile("/Users/tietang/my/gitcode/r_app/src/github.com/tietang/spring-boot-collector/app/config.ini")
    c.Start()

}
