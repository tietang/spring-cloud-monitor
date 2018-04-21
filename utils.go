package collector

import "strings"

func toDatabaseName(name string) string {
    return toInfluxName(name)

}

func toInfluxName(key string) string {
    //替换-为_
    name := strings.Replace(key, "-", "_", -1)
    //替换.为_
    name = strings.Replace(name, ".", "_", -1)
    return name
}
func toMeasurementName(key string) string {
    //替换CONF_METRICS_GROUP_PREFEX为_
    name := strings.Replace(key, CONF_METRICS_GROUP_PREFEX, "", 1)
    return toInfluxName(name)
}
func toTagOrFieldName(key string, prefix string) string {
    if strings.Index(key, ".") == -1 {
        return key
    } else {
        //log.Info(key, prefix)
        name := strings.Replace(key, prefix, "", -1)
        name = strings.Replace(name, ".", "_", -1)
        return name
    }
}
