#! /usr/bin/env sh
GOPATH=`pwd` go get -v -u all
GOPATH=`pwd` go install -v all
