#!/bin/bash
# Build script for Max Bot Docker image
# Reads JOOMLA_* variables from .env file and passes them as build args

set -e

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Error: .env file not found!"
    echo "Please create .env file with required variables."
    exit 1
fi

# Load variables from .env
set -a
source .env
set +a

# Check required variables
if [ -z "$JOOMLA_SITE_URL" ]; then
    echo "Warning: JOOMLA_SITE_URL not set, using default"
    export JOOMLA_SITE_URL="https://plk32.ru"
fi

if [ -z "$JOOMLA_ADMIN_URL" ]; then
    echo "Warning: JOOMLA_ADMIN_URL not set, using default"
    export JOOMLA_ADMIN_URL="https://plk32.ru/administrator"
fi

if [ -z "$JOOMLA_USERNAME" ] || [ -z "$JOOMLA_PASSWORD" ]; then
    echo "Error: JOOMLA_USERNAME and JOOMLA_PASSWORD must be set in .env"
    exit 1
fi

# Build Docker image with build args
echo "Building Docker image..."
echo "  JOOMLA_SITE_URL: $JOOMLA_SITE_URL"
echo "  JOOMLA_ADMIN_URL: $JOOMLA_ADMIN_URL"
echo "  JOOMLA_USERNAME: $JOOMLA_USERNAME"
echo "  JOOMLA_PASSWORD: ${JOOMLA_PASSWORD:0:3}***"
echo "  JOOMLA_API_TOKEN: ${JOOMLA_API_TOKEN:0:3}***"

docker build \
    --build-arg JOOMLA_SITE_URL="$JOOMLA_SITE_URL" \
    --build-arg JOOMLA_ADMIN_URL="$JOOMLA_ADMIN_URL" \
    --build-arg JOOMLA_USERNAME="$JOOMLA_USERNAME" \
    --build-arg JOOMLA_PASSWORD="$JOOMLA_PASSWORD" \
    --build-arg JOOMLA_API_TOKEN="${JOOMLA_API_TOKEN:-}" \
    -t max-bot \
    .

echo ""
echo "Build completed successfully!"
echo ""
echo "To run the container:"
echo "  docker run -d --env-file .env --name max-bot-container max-bot"
echo ""
echo "Or use docker-compose:"
echo "  docker-compose up -d"
