
# spring-cloud-monitor

[![Build Status](https://travis-ci.org/tietang/spring-cloud-monitor.svg?branch=master)](<https://travis-ci.org/tietang/spring-cloud-monitor>)
[![GoDoc Documentation](http://godoc.org/github.com/tietang/spring-cloud-monitor?status.png)](<https://godoc.org/github.com/tietang/spring-cloud-monitor>)
[![Sourcegraph](https://sourcegraph.com/github.com/tietang/spring-cloud-monitor/-/badge.svg)](https://sourcegraph.com/github.com/tietang/spring-cloud-monitor?badge)
[![GitHub release](https://img.shields.io/github/release/tietang/spring-cloud-monitor.svg)](https://github.com/tietang/spring-cloud-monitor/releases)

 
一个基于eureka服务发现对微服务应用JVM和WEB请求指标、微服务运行状态的监控，并通过图形化来展示的小型轻量级监控系统。

## 特性

- 服务发现
- influxdb读写支持基于微服务名称的HASH算法分片
- grafana支持influxdb分片导航
- 微服务运行状态图
- jvm监控图

## 架构

![](<doc/imgs/health-check-13.png>)

## 安装

### 依赖安装
spring-cloud-monitor安装依赖如下服务，请按照官网安装文档安装：

- grafana [https://grafana.com/get](https://grafana.com/get)  
- influxdb  [https://portal.influxdata.com/downloads](https://portal.influxdata.com/downloads)
- redis  [https://redis.io/download](https://redis.io/download)

### spring-cloud-monitor程序安装


#### 编译安装

安装golang >1.9.x,执行build.sh编译.

>$ cd /path/to/spring-cloud-monitor/app
>
>$ ./build.sh


#### 微服务健康状态图

微服务监控状态Dashboard中直接从redis（单机版直接从内存中）取出微服务列表，再根据微服务获取实例健康状态值。在Dashboard界面上设计了10种渐变颜色来说明服务的健康状态如下图所示：

![{ :100 }](<doc/imgs/health-check-6.png>) 

从红色到绿色，分别来说明健康状态，红色表示所有服务已经不可用了，绿色代表所有服务可用，中间态表示部分服务可用，部分服务不可用。变红表示开始有些实例不可用了，变绿表示不可用实例开始恢复了。

并使用3个区间来分别表示1分钟、5分钟、15分钟服务健康状态，如下所示：

![](<doc/imgs/health-check-5.png>)

并计算所有实例1分钟的健康状态值的平均数来做为微服务的健康状态，如下所示，有一个实例已经挂了：

![](<doc/imgs/health-to-down-1.png>)

如下所示，所有实例已经挂了：


![](<doc/imgs/health-to-down-all.png>)

如下所示，其中一个实例刚刚在1分钟内down了：

![](<doc/imgs/health-to-down1.png>)

如下所示，其中一个实例已经挂了快5分钟了：

![](<doc/imgs/health-to-down5.png>)


如下所示，其中一个实例已经挂了快5分钟了，但已经开始恢复了：

![](<doc/imgs/health-to-down5-to-up.png>)
![](<doc/imgs/health-to-down5-to-up-2.png>)

如下所示，其中一个已经挂了的实例，在1分钟前恢复了：


![](<doc/imgs/health-check-8.png>)

如下所示，如果应用未开放`/health`（或者无法连接）,显示会灰色：

![](<doc/imgs/health-no.png>)


## 监控图展示：

##### 总体

![](<doc/imgs/d-1.png>)
![](<doc/imgs/d-2.png>)
![](<doc/imgs/d-3.png>)

##### 平均响应时间：
![](<doc/imgs/d-res-1.png>)

##### 每分钟调用次数（QPM）：
![](<doc/imgs/d-res-2.png>)

##### 部署实例状态图：

正常：
![](<doc/imgs/d-status-host-0.png>)

有2个实例不在线：
![](<doc/imgs/d-status-host-1.png>)

实例状态趋势图：

![](<doc/imgs/d-i-status-1.png>)

##### CPU处理平均核数和平均使用率：
![](<doc/imgs/dashboard_cpu.png>)


