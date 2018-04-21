#!/usr/bin/env bash

#go build metrics_collector.go
nohup ./metrics_collector > mc.log 2>&1 &
#
