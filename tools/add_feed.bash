#!/bin/bash

echo $0
echo $1

curl -v -X POST -H 'Content-Type: application/json' -d "{\"url\": \"$1\"}" http://localhost:8081/api/feeds
