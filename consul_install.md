
https://www.consul.io/downloads.html
https://releases.hashicorp.com/consul/1.0.2/consul_1.0.2_linux_amd64.zip?_ga=2.114159943.1510528008.1515463248-766885714.1494220877


#!/bin/bash
BASEDIR=`dirname $0`/../../..
nohup ./consul agent -server -bootstrap-expect 1 -advertise 192.168.15.162 -data-dir ./data -config-dir=./config -client 0.0.0.0 -ui > ./consul.log 2>&1 &
# -ui-dir ${BASEDIR}/src/test/resources/consul_ui