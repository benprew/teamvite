#!/bin/bash

docker run --rm -v "$PWD":/usr/src/app -w /usr/src/app golang:1.17-bullseye go build -v
