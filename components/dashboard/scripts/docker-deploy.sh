#!/bin/bash

# Docker Deployment Script for Language Operator Dashboard
# Based on Prisma Docker best practices

set -e

echo "🚀 Starting Docker deployment..."

# Build the Docker image
echo "📦 Building Docker image..."
docker build -t language-operator-dashboard:latest .

# Run database migrations in a temporary container
echo "📊 Running database migrations..."
docker run --rm \
  --network host \
  -e DATABASE_URL="${DATABASE_URL}" \
  language-operator-dashboard:latest \
  sh -c "npx prisma db push && npx prisma generate"

echo "✅ Database migrations completed"

# Start the application
echo "🌟 Starting application..."
docker run -d \
  --name language-operator-dashboard \
  --network host \
  -e DATABASE_URL="${DATABASE_URL}" \
  -e NEXTAUTH_SECRET="${NEXTAUTH_SECRET}" \
  -e NEXTAUTH_URL="${NEXTAUTH_URL}" \
  -e GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID:-}" \
  -e GOOGLE_CLIENT_SECRET="${GOOGLE_CLIENT_SECRET:-}" \
  -e GITHUB_CLIENT_ID="${GITHUB_CLIENT_ID:-}" \
  -e GITHUB_CLIENT_SECRET="${GITHUB_CLIENT_SECRET:-}" \
  --restart unless-stopped \
  language-operator-dashboard:latest

echo "✅ Language Operator Dashboard deployed successfully!"
echo "🌐 Application running at: ${NEXTAUTH_URL:-http://localhost:3000}"