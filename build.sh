#!/bin/bash

GOOS=linux go build -x -ldflags "-s -w" -o xfbbroker ./exe/main.go
