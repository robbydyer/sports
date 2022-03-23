#!/bin/bash
set -euo pipefail

gcloud_auth() {
  gcloud init
  gcloud auth list
  gcloud auth login --update-adc
}

install_gcloud() {
  if which gcloud; then
    return
  fi

  echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list

  curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo tee /usr/share/keyrings/cloud.google.gpg

  sudo apt-get update && sudo apt-get install google-cloud-cli
}

install_gcloud
gcloud_auth
