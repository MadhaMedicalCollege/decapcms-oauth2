#!/bin/bash
set -e

# Configuration
AWS_REGION="ap-south-1" # Change to your preferred region
ECR_REPOSITORY_NAME="madha-github-oauth2"
LAMBDA_FUNCTION_NAME="madha-github-oauth2"

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Get GitHub SHA from origin/main
git fetch origin
ORIGIN_SHA=$(git rev-parse --short origin/main)
echo "Using origin/main SHA: ${ORIGIN_SHA}"

# Create ECR repository if it doesn't exist
echo "Checking ECR repository if it exist..."
aws ecr describe-repositories --repository-names ${ECR_REPOSITORY_NAME} --region ${AWS_REGION}

# Login to ECR
echo "Logging in to ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Build the Docker image
echo "Building Docker image..."
docker build -t ${ECR_REPOSITORY_NAME}:${ORIGIN_SHA} .

# Tag the image with SHA and latest
echo "Tagging image with origin SHA: ${ORIGIN_SHA}..."
docker tag ${ECR_REPOSITORY_NAME}:${ORIGIN_SHA} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}:${ORIGIN_SHA}

# Push the images to ECR
echo "Pushing images to ECR..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}:${ORIGIN_SHA}

echo "Image pushed successfully with tags: ${ORIGIN_SHA}"
