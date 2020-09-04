#!/bin/bash
set -euo pipefail

# Get OS data.
source /etc/os-release
ARCH=$(uname -m)

# Colorful output.
function greenprint {
    echo -e "\033[1;32m${1}\033[0m"
}

# Enable EPEL on RHEL
if [[ $ID == rhel ]] && ! rpm -q epel-release; then
    greenprint "📦 Setting up EPEL repository"
    curl -Ls --retry 5 --output /tmp/epel.rpm \
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    sudo rpm -Uvh /tmp/epel.rpm
fi

greenprint "Installing required packages"
sudo dnf -y install \
    dnsmasq \
    koji \
    podman

greenprint "Starting containers"
sudo ./internal/upload/koji/run-koji-container.sh start

greenprint "Adding generated CA cert"
sudo cp \
    /tmp/osbuild-composer-koji-test/ca-crt.pem \
    /etc/pki/ca-trust/source/anchors/koji-ca-crt.pem
sudo update-ca-trust

greenprint "Testing Koji"
koji --server=http://localhost/kojihub --user=osbuild --password=osbuildpass --authtype=password hello

greenprint "Pushing compose to Koji"
./test/image-tests/koji-compose.py

greenprint "Stopping containers"
sudo ./internal/upload/koji/run-koji-container.sh stop

greenprint "Removing generated CA cert"
sudo rm \
    /etc/pki/ca-trust/source/anchors/koji-ca-crt.pem
sudo update-ca-trust
