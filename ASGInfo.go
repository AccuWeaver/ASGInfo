package main

import (
	"ASGInfo/autoscalinghandler"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
)

// main - entry point for the lambda function
func main() {
	lambda.Start(cfn.LambdaWrap(autoscalinghandler.ASGInfoLambda))
}
