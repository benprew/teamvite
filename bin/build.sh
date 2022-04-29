#!/bin/bash

[[ -e /usr/local/musl/bin/musl-gcc ]] || { echo "ERROR: musl not installed"; exit 1; }

# to install musl libc:
# 1. download latest release from https://www.musl-libc.org/download.html
# 2. tar zxvf <file>
# 3. ./configure --disable-shared && make && sudo make install
# - by default musl installs into /usr/local/musl to avoid libc conflicts

CGO_ENABLED=1 CC=/usr/local/musl/bin/musl-gcc \
              go build \
              -ldflags="-extldflags=-static"
