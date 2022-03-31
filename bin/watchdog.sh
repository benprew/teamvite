#!/bin/bash

cleanup () {
    kill $PID
}

trap cleanup EXIT

while true; do
    $@ &
    PID=$!
    inotifywait -e modify -e create -e move -e attrib $1
    kill $PID
done
