#!/bin/bash
set -euo pipefail

set -x

if ! which jq; then
  sudo apt-get install -y jq
fi

ARCH="$(uname -m)"

latesturl="https://api.github.com/repos/robbydyer/sports/releases"

tmp="$(mktemp -d /tmp/sportsinstall.XXXX)"
trap "rm -rf ${tmp}" EXIT

cd "${tmp}"
download="$(curl -s "${latesturl}" | jq -r 'map(select(.prerelease)) | first | .assets | first | .browser_download_url' | sed "s,aarch64,${ARCH},g" | sed "s,armv6l,${ARCH},g" | sed "s,armv7l,${ARCH},g" )"

wget -qi "${download}"

sudo dpkg -i --force-confdef --force-confold sportsmatrix*_${ARCH}.deb

sudo systemctl enable sportsmatrix
sudo systemctl restart sportsmatrix
