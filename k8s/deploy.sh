#!/bin/bash
set -e

NAMESPACE=globeco
POSTGRES_STS=globeco-fix-engine-postgresql

# Deploy PostgreSQL StatefulSet and Service
kubectl apply -f k8s/postgresql-deployment.yaml

echo "Waiting for PostgreSQL StatefulSet to be ready..."
# Wait for the StatefulSet to have at least 1 ready replica
until kubectl -n "$NAMESPACE" get statefulset "$POSTGRES_STS" -o jsonpath='{.status.readyReplicas}' | grep -q '^1$'; do
  echo "  ...still waiting for PostgreSQL to be ready..."
  sleep 5
done

echo "PostgreSQL StatefulSet is ready. Deploying Fix Engine."

# Deploy application Deployment and Service
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
echo "Deployment complete."
