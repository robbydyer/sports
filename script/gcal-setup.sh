#!/bin/bash
set -euo pipefail

gcloud_auth() {
  # We use sudo here, because the root user is what runs the matrix
  sudo gcloud init --no-browser --skip-diagnostics
  sudo gcloud auth list
  sudo gcloud auth application-default login
}

install_gcloud() {
  if which gcloud &> /dev/null; then
    return
  fi

  echo "=> Installing gcloud"
  echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list

  curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo tee /usr/share/keyrings/cloud.google.gpg

  sudo apt-get update && sudo apt-get install google-cloud-cli
}

install_gcloud
gcloud_auth
