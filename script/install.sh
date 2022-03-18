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
  echo "This architecture is deprecated! The last supported version is v0.0.83"
  wget -q "https://github.com/robbydyer/sports/releases/download/v0.0.83/sportsmatrix-0.0.83_armv6l.deb"
  ;;
*)
  echo "Unsupported architecture '${ARCH}'"
  exit 1
  ;;
esac


sudo dpkg -i --force-confdef --force-confold sportsmatrix*_${ARCH}.deb

sudo systemctl enable sportsmatrix
sudo systemctl restart sportsmatrix
