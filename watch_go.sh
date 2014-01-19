#! /usr/bin/env sh
./build.sh

GOPATH=`pwd`

./bin/cmo &

inotifywait -m -r -e close_write src/app/ | while read line
do
	go build -o bin/cmo app && pkill cmo && ./bin/cmo &
done

