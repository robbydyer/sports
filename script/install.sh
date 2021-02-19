#!/bin/bash
set -euo pipefail

set -x

ARCH="$(uname -m)"

latesturl="http://api.github.com/robbydyer/sports/releases/latest"

tmp="$(mktemp -d /tmp/sportsinstall.XXXX)"
trap "rm -rf ${tmp}" EXIT

cd "${tmp}"
curl -s "${latesturl}" | grep browser_download_url | grep deb | cut -d: -f2,3 | tr -d \" | wget -qi -

sudo dpkg -i sportsmatrix*_${ARCH}.deb