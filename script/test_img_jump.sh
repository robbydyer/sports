#!/bin/bash

set -ex
host="matrix.local"
path="imageboard.v1.ImageBoard"

curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"name":"tigger.png"}' \
  "http://${host}/${path}/Jump"

