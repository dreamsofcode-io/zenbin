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
declare -a PORTS=("3000" "3001" "3002")

# Check if the binary exists
if [ ! -f "${RELEASES_DIR}/${BINARY_NAME}" ]; then
  echo "Binary ${BINARY_NAME} not found in ${RELEASES_DIR}"
  exit 1
fi

# Keep a reference to the previous binary from the symlink
if [ -L "${DEPLOY_BIN}" ]; then
  PREVIOUS=$(readlink -f $DEPLOY_BIN)
  echo "Current binary is ${PREVIOUS}, saved for rollback."
else
  echo "No symbolic link found, no previous binary to backup."
  PREVIOUS=""
fi

rollback_deployment() {
  if [ -n "$PREVIOUS" ]; then
    echo "Rolling back to previous binary: ${PREVIOUS}"
    ln -sfn "${PREVIOUS}" "${DEPLOY_BIN}"
  else
    echo "No previous binary to roll back to."
  fi

  # Restart all services with the previous binary
  for port in "${PORTS[@]}"; do
    SERVICE="${SERVICE_NAME}@${port}.service"
    echo "Restarting $SERVICE..."
    sudo systemctl restart $SERVICE
  done

  echo "Rollback completed."
}

# Copy the binary to the deployment directory
echo "Promoting ${BINARY_NAME} to ${DEPLOY_BIN}..."
ln -sf "${RELEASES_DIR}/${BINARY_NAME}" "${DEPLOY_BIN}"

WAIT_TIME=5
restart_service() {
  local port=$1
  local SERVICE="${SERVICE_NAME}@${port}.service"
  echo "Restarting ${SERVICE}..."

  # Restart the service
  if ! sudo systemctl restart "$SERVICE"; then
    echo "Error: Failed to restart ${SERVICE}. Rolling back deployment."

    # Call the rollback function
    rollback_deployment
    exit 1
  fi

  # Wait a few seconds to allow the service to fully start
  echo "Waiting for ${SERVICE} to fully start..."
  sleep $WAIT_TIME

  # Check the status of the service
  if ! systemctl is-active --quiet "${SERVICE}"; then
    echo "Error: ${SERVICE} failed to start correctly. Rolling back deployment."

    # Call the rollback function
    rollback_deployment
    exit 1
  fi

  echo "${SERVICE}.service restarted successfully."
}

for port in "${PORTS[@]}"; do
  restart_service $port
done

echo "Deployment completed successfully."
