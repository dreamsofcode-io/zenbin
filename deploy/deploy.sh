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
DEPLOY_BIN="/home/deploy/production/zenbin"
SERVICE_NAME="zenbin"
BINARY_NAME="zenbin-${COMMIT_HASH}"

# Check if the binary exists
if [ ! -f "${RELEASES_DIR}/${BINARY_NAME}" ]; then
  echo "Binary ${BINARY_NAME} not found in ${RELEASES_DIR}"
  exit 1
fi

# Copy the binary to the deployment directory
echo "Promoting ${BINARY_NAME} to ${DEPLOY_BIN}..."
ln -sf "${RELEASES_DIR}/${BINARY_NAME}" "${DEPLOY_BIN}"

for port in 3000 3001 3002; do
  # Restart the service
  SERVICE="${SERVICE_NAME}@${port}.service"
  echo "Restarting the ${SERVICE} service..."
  sudo systemctl restart ${SERVICE}
done

echo "Deployment completed successfully."
