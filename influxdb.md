
##### 2、InfluxDB下载和安装

使用非Root用户账号登录，将InfluxDB下载到指定目录下

``` 
// 下载InfluxDB
$ wget https://dl.influxdata.com/influxdb/releases/influxdb-1.2.4_linux_amd64.tar.gz

// 解压InfluxDB
$ tar -zxvf influxdb-1.2.4_linux_amd64.tar.gz

// 删除安装包
$ rm -rf influxdb-1.2.4_linux_amd64.tar.gz
```

https://dl.influxdata.com/influxdb/releases/influxdb-1.4.2_linux_amd64.tar.gz
https://dl.influxdata.com/chronograf/releases/chronograf-1.4.0.0_linux_amd64.tar.gz

##### 3、InfluxDB配置信息

``` 
[meta]
  # Where the metadata/raft database is stored
  dir = "influxdb目录/var/lib/influxdb/meta"

[data]
  # The directory where the TSM storage engine stores TSM files.
  dir = "influxdb目录/var/lib/influxdb/data"

  # The directory where the TSM storage engine stores WAL files.
  wal-dir = "influxdb目录/var/lib/influxdb/wal"

[admin]
  # <=1.2.x 支持 >=1.3请下载chronograf
  # Determines whether the admin service is enabled.
  enabled = true

  # The default bind address used by the admin service.
  bind-address = ":8083"

  # Whether the admin service should use HTTPS.
  # https-enabled = false

  # The SSL certificate used when HTTPS is enabled.
  # https-certificate = "/etc/ssl/influxdb.pem"
  
[http]
  # Determines whether HTTP endpoint is enabled.
  enabled = true

  # The bind address used by the HTTP service.
  bind-address = ":8086"

  # Determines whether HTTP authentication is enabled.
  auth-enabled = true
```

##### 4、InfluxDB启动和停止

``` 
// 启动InfluxDB
$ nohup ./influxd -config ../../etc/influxdb/influxdb.conf > influxdb.log &

// 查看InfluxDB进程
$ ps -ef | grep influxdb

// 停止InfluxDB
$ kill InfluxDB进程号
```
