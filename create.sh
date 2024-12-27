#!/usr/bin/env bash
# To get the session, export the AWS_PROFILE and AWS_REGION and then run ../../../ci/awsSetup.sh
account_id=$( aws sts get-caller-identity | jq -r '.Account')
echo "Deploying to account ${account_id}"

aws iam create-role --role-name ASGInfo --assume-role-policy-document file://./trust-policy.json

aws iam attach-role-policy --role-name ASGInfo --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam attach-role-policy --role-name ASGInfo --policy-arn arn:aws:iam::aws:policy/AmazonEC2FullAccess
aws iam attach-role-policy --role-name ASGInfo --policy-arn arn:aws:iam::aws:policy/AWSCloudFormationReadOnlyAccess

env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /tmp/main ASGInfo.go
zip -j /tmp/main.zip /tmp/main

aws lambda create-function --function-name ASGInfo \
    --runtime go1.x \
    --role arn:aws:iam::${account_id}:role/ASGInfo \
    --handler main --zip-file fileb:///tmp/main.zip \
    --timeout 300
