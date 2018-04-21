#!/usr/bin/env bash
VERSION=0.1
cpath=`pwd`
echo $cpath
PROJECT_PATH=${cpath%src*} #从右向左截取第一个 src 后的字符串
echo ${PROJECT_PATH}


export GOPATH=$GOPATH:${PROJECT_PATH}

SOURCE_FILE_NAME=metrics_collector
TARGET_FILE_NAME=metrics_collector

path=`pwd`

echo $path
echo ${path%src*} #从右向左截取第一个 src 后的字符串


build(){
   echo $GOOS $GOARCH
   env  GOOS=${GOOS} GOARCH=${GOARCH}  go build -o ${TARGET_FILE_NAME}_${GOOS}_${GOARCH}-${VERSION}${EXT} -v ${SOURCE_FILE_NAME}.go
   mv ${TARGET_FILE_NAME}_${GOOS}_${GOARCH}-${VERSION}${EXT} ${TARGET_FILE_NAME}${EXT}
   if [ ${GOOS} == "windows" ]; then
       zip ${TARGET_FILE_NAME}_${GOOS}_${GOARCH}-${VERSION}.zip ${TARGET_FILE_NAME}${EXT} conf.ini ./assets/ ./public/
   else
       tar -czvf ${TARGET_FILE_NAME}_${GOOS}_${GOARCH}-${VERSION}.tar.gz ${TARGET_FILE_NAME}${EXT} conf.ini  ./assets/ ./public/ run.sh stop.sh
   fi
   mv  ${TARGET_FILE_NAME}${EXT} ${TARGET_FILE_NAME}_${GOOS}_${GOARCH}-${VERSION}${EXT}
}

CGO_ENABLED=0
# linux
GOOS=linux
GOARCH=amd64
EXT=
build

# mac osx
GOOS=darwin
GOARCH=amd64
build
# windows
GOOS=windows
GOARCH=amd64
EXT=.exe

build

GOARCH=386
build
