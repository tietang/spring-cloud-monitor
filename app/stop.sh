#!/usr/bin/env bash

ps -ef|grep metrics_collector|awk '{print $2}'|while read pid
        do
                kill -9 $pid
        done