#!/bin/bash

# Exit on error
set -e

# Check if commit hash is passed as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 <commit-hash>"
  exit 1
fi

COMMIT_HASH=$1
RELEASES_DIR="/home/deploy/releases"
DEPLOY_DIR="/home/deploy/production/zenbin"
SERVICE_NAME="zenbin.service"
BINARY_NAME="zenbin-${COMMIT_HASH}"

# Check if the binary exists
if [ ! -f "${RELEASES_DIR}/${BINARY_NAME}" ]; then
  echo "Binary ${BINARY_NAME} not found in ${RELEASES_DIR}"
  exit 1
fi

# Copy the binary to the deployment directory
echo "Promoting ${BINARY_NAME} to ${DEPLOY_DIR}..."
ln -sf "${RELEASES_DIR}/${BINARY_NAME}" "${DEPLOY_DIR}"

# Restart the service
echo "Restarting the ${SERVICE_NAME} service..."
systemctl restart ${SERVICE_NAME}

echo "Deployment completed successfully."
