#!/bin/bash
set -ex

rm -rf bin/
docker build --rm -t https .
docker run --rm -v $PWD:/go/src/app https
