#!/bin/sh
# Adapted from h2non's
# https://github.com/h2non/imaginary/blob/master/benchmark.sh

set -xe

port=8000
root=fixtures
image=lena.jpg
cache=.benchmark.db

# Install Vegeta
# go get -u github.com/tsenart/vegeta

GOPATH=`pwd`
PATH=$PATH:$GOPATH/bin

rm -f $cache
make
./bin/iiif -port $port -root $root & > /dev/null 2>&1
pid=$!

suite() {
  echo "$1 --------------------------------------"
  echo "GET http://localhost:$port/$image/$2" \
    | vegeta attack \
        -duration=30s \
        -rate=50 \
    | ./bin/vegeta report
  sleep 1
}

# Run suites
suite "info" "info.json"
#suite "square" "square/full/0/default.jpg"
#suite "max" "full/max/0/default.jpg"
#suite "rotate" "full/max/180/default.jpg"
#suite "flip" "full/max/!0/default.jpg"
#suite "gray" "full/full/0/gray.jpg"

# Cleanup
rm -f $cache
# Kill the server
kill -9 $pid
