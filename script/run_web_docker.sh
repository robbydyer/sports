#!/bin/bash
# Run this on the Raspberry-pi to start the container for the web manager
set -euo pipefail

ROOT="$(dirname $( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd ))"
LISTENPORT=8081

if ! dpkg -l | grep docker.io; then
  apt-get update
  apt-get install -y docker.io
fi

source "${ROOT}/script/common"
IMG="matrixweb:$(getsha ${ROOT}/Dockerfile.web)"

if [ "$(docker images -q ${IMG})" = "" ]; then
  docker build -f "${ROOT}/Dockerfile.web" -t ${IMG} "${ROOT}"
fi

c="matrixweb"
if docker inspect "${c}" &> /dev/null; then
  echo "=> Killing container ${c}"
  docker kill "${c}" || true
fi
set +e
docker rm "${c}"
set -e

docker run -d \
  --name "${c}" \
  --restart=unless-stopped \
  --publish ${LISTENPORT}:80 \
  -v "${ROOT}/web/build:/var/www/html" \
  -e APACHE_RUN_USER=root \
  ${IMG} \
  bash -cex "service apache2 start ; while true; do sleep 6000; done"
