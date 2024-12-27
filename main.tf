provider "aws" {
  # Set the profile to the workspace name
  profile = terraform.workspace
  region  = "us-east-1"  # Replace with your desired region
}

# Lambda function name
variable "lambda_function_name" {
  description = "The name of the Lambda function"
  type        = string
  default     = "ASGInfo"
}

# IAM role name
variable "lambda_role_name" {
  description = "The name of the IAM role for the Lambda function"
  type        = string
  default     = "ASGInfo"
}

# Create the IAM role for the Lambda function
resource "aws_iam_role" "lambda_role" {
  name = var.lambda_role_name

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# Attach the AmazonEC2FullAccess policies to the IAM role
resource "aws_iam_role_policy_attachment" "amazon_ec2_full_access_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
}

# Attach the AWSLambdaBasicExecutionRole policies to the IAM role
resource "aws_iam_role_policy_attachment" "aws_lambda_basic_execution_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Attach the AWSLambdaRole policies to the IAM role
resource "aws_iam_role_policy_attachment" "aws_lambda_role_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaRole"
}


# null resource to build the Lambda function
resource "null_resource" "build_lambda" {
  provisioner "local-exec" {
    command = <<EOT
    {
      ERROR_FILE=$(mktemp)
      echo "Temporary file created at: $ERROR_FILE"
      OUTPUT=$(GOOS=linux GOARCH=amd64 go build -o ASGInfo ASGInfo.go 2> $ERROR_FILE)
      RET_CODE=$?
      ERROR=$(cat $ERROR_FILE)
      rm $ERROR_FILE

      if [ $RET_CODE -ne 0 ]; then
        echo '{"stdout": "'$OUTPUT'", "stderr": "'$ERROR'", "return_code": "'$RET_CODE'"}' > build_output.json
        exit 1
      fi

      OUTPUT=$(chmod +x ASGInfo bootstrap 2>> $ERROR_FILE)
      RET_CODE=$?
      ERROR=$(cat $ERROR_FILE)
      rm $ERROR_FILE

      if [ $RET_CODE -ne 0 ]; then
        echo '{"stdout": "'$OUTPUT'", "stderr": "'$ERROR'", "return_code": "'$RET_CODE'"}' > build_output.json
        exit 1
      fi

      OUTPUT=$(zip ASGInfo.zip ASGInfo bootstrap 2>> $ERROR_FILE)
      RET_CODE=$?
      ERROR=$(cat $ERROR_FILE)
      rm $ERROR_FILE

      if [ $RET_CODE -ne 0 ]; then
        echo '{"stdout": "'$OUTPUT'", "stderr": "'$ERROR'", "return_code": "'$RET_CODE'"}' > build_output.json
        exit 1
      fi

      HASH=$(openssl dgst -sha256 -binary ASGInfo.zip | openssl enc -base64)
      echo '{"stdout": "'$OUTPUT'", "stderr": "'$ERROR'", "return_code": "'$RET_CODE'", "hash": "'$HASH'", "return_code": "0"}' > build_output.json
    }
  EOT
  }

  triggers = {
    always_run = timestamp()
  }
}

# Data source to read the build output
data "external" "build_output" {
  program = ["sh", "-c", "cat build_output.json"]

  depends_on = [null_resource.build_lambda]
}

# Create the Lambda function
resource "aws_lambda_function" "ASGInfo" {
  function_name = var.lambda_function_name
  role          = aws_iam_role.lambda_role.arn
  handler       = "ASGInfo"
  runtime = "provided.al2023"  # Use custom runtime with Amazon Linux 2023
  filename      = "ASGInfo.zip"

  source_code_hash = data.external.build_output.result[
  "hash"
  ]

  depends_on = [
    null_resource.build_lambda
  ]
}

# Create or update the log group for the Lambda function
resource "aws_cloudwatch_log_group" "lambda_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.ASGInfo.function_name}"
  retention_in_days = 1
}

# Output the build message
output "build_message" {
  value = data.external.build_output.result
}

# Create the Lambda function event invoke config
resource "aws_lambda_function_event_invoke_config" "ASGInfo_event_invoke_config" {
  function_name = aws_lambda_function.ASGInfo.function_name

  maximum_retry_attempts       = 2
  maximum_event_age_in_seconds = 60
}