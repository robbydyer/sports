#!/bin/bash
set -euo pipefail

set -x

ARCH="$(uname -m)"

latesturl="https://api.github.com/repos/robbydyer/sports/releases/latest"

tmp="$(mktemp -d /tmp/sportsinstall.XXXX)"
trap "rm -rf ${tmp}" EXIT

cd "${tmp}"
curl -s "${latesturl}" | grep browser_download_url | grep deb | cut -d: -f2,3 | tr -d \" | wget -qi -

sudo dpkg -i --force-confdef --force-confold sportsmatrix*_${ARCH}.deb

sudo systemctl enable sportsmatrix
sudo systemctl restart sportsmatrix
