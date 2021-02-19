#!/bin/bash
set -euo pipefail

ROOT="$(dirname $( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd ))"
ARCH="$(uname -m)"

sudo apt-get install -y build-essential

cd "${ROOT}"
script/build.local

tmp="$(mktemp -d /tmp/sportsbuild.XXXX)"
echo "Build Dir: ${tmp}"

d="sportsmatrix-${VERSION}_${ARCH}"

mkdir "${tmp}/${d}" 
cd "${tmp}/${d}"

mkdir -p DEBIAN etc/systemd/system usr/local/bin etc/logrotate.d

cat <<EOF > DEBIAN/control
Package: sportsmatrix
Version: ${VERSION}
Section: custom
Priority: optional
Architecture: all
Essential: no
Maintainer: https://github.com/robbydyer/sports
Description: Live sports driver for RGB LED matrix
EOF

cat <<EOF > etc/systemd/system/sportsmatrix.service
[Unit]
Description=Sportsmatrix
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
User=root
ExecStart=/usr/local/bin/sportsmatrix run -f /var/log/sportsmatrix.log

[Install]
WantedBy=multi-user.target
EOF

cat <<EOF > etc/logrotate.d/sportsmatrix
/var/log/sportsmatrix.log
{
        rotate 3
        daily
        missingok
        notifempty
        delaycompress
        compress
}
EOF

cp "${ROOT}/sportsmatrix.bin" usr/local/bin/sportsmatrix
cp "${ROOT}/sportsmatrix.conf.example" etc/sportsmatrix.conf

cd "${tmp}"
dpkg-deb --build "${d}"

mv "${d}.deb" "${ROOT}/"

cd "${ROOT}"
rm -rf "${tmp}"