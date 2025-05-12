#!/bin/bash
set -e

# Configuration
AWS_REGION="ap-south-1" # Change to your preferred region
ECR_REPOSITORY_NAME="madha-github-oauth2"
LAMBDA_FUNCTION_NAME="madha-github-oauth2"

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Get GitHub SHA
GIT_SHA=$(git rev-parse --short HEAD)

# Create ECR repository if it doesn't exist
echo "Checking ECR repository if it exist..."
aws ecr describe-repositories --repository-names ${ECR_REPOSITORY_NAME} --region ${AWS_REGION}

# Login to ECR
echo "Logging in to ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Build the Docker image
echo "Building Docker image..."
docker build -t ${ECR_REPOSITORY_NAME}:${GIT_SHA} .

# Tag the image
echo "Tagging image with SHA: ${GIT_SHA}..."
docker tag ${ECR_REPOSITORY_NAME}:${GIT_SHA} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}:${GIT_SHA}

# Push the image to ECR
echo "Pushing image to ECR..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}:${GIT_SHA}
