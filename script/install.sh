#!/bin/bash
set -euo pipefail

set -x

ARCH="$(uname -m)"

tmp="$(mktemp -d /tmp/sportsinstall.XXXX)"
trap "rm -rf ${tmp}" EXIT

cd "${tmp}"

case "${ARCH}" in
aarch64|armv7l)
  echo "Installing sportsmatrix for ${ARCH}"
  latesturl="https://api.github.com/repos/robbydyer/sports/releases/latest"

  curl -s "${latesturl}" | grep browser_download_url | grep deb | cut -d: -f2,3 | tr -d \" | wget -qi -
  ;;
armv6l)
  echo "Sorry, this architecture is not supported. You need a newer Pi with an armv7 processor- Pi 3, 4 or Pi Zero 2"
  ;;
*)
  echo "Unsupported architecture '${ARCH}'"
  exit 1
  ;;
esac


sudo dpkg -i --force-confdef --force-confold sportsmatrix*_${ARCH}.deb

sudo systemctl enable sportsmatrix
sudo systemctl restart sportsmatrix
