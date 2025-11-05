#!/bin/bash

# Multi-architecture Docker build script
set -e

IMAGE_NAME=${IMAGE_NAME:-kasbench/globeco-fix-engine}
TAG=${TAG:-latest}
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64}

echo "Building multi-architecture image: ${IMAGE_NAME}:${TAG}"
echo "Platforms: ${PLATFORMS}"

# Create and use buildx builder if it doesn't exist
docker buildx create --name multiarch-builder --use 2>/dev/null || docker buildx use multiarch-builder

# Build and push multi-architecture image
docker buildx build \
  --platform ${PLATFORMS} \
  -t ${IMAGE_NAME}:${TAG} \
  -t ${IMAGE_NAME}:latest \
  --push \
  .

echo "Multi-architecture build completed successfully!"
