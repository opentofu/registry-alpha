data "aws_iam_policy_document" "assume_lambda_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type = "Service"
      identifiers = [
        "apigateway.amazonaws.com",
        "lambda.amazonaws.com"
      ]
    }
  }
}

data "aws_iam_policy_document" "github_api_token_secrets_iam_policy" {
  statement {
    effect = "Allow"
    actions = [
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      aws_secretsmanager_secret.github_api_token.arn,
    ]
  }
}

resource "aws_iam_policy" "lambda_secrets_policy" {
  name        = "${var.domain_name}-RegistryLambdaSecretsPolicy"
  description = "Policy for lambda to pull its secrets"
  policy      = data.aws_iam_policy_document.github_api_token_secrets_iam_policy.json
}

resource "aws_iam_role_policy_attachment" "lambda_secrets_policy_attachment" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.lambda_secrets_policy.arn
}

resource "aws_iam_role" "lambda" {
  name               = "${var.domain_name}-RegistryLambdaRole"
  description        = "Role for the registry to assume lambda"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role.json
}

data "aws_iam_policy_document" "allow_lambda_logging" {
  # Allow CloudWatch logging
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:DescribeLogGroups",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents"
    ]

    resources = [
      "arn:aws:logs:*:*:*",
    ]
  }

  # Allow X-Ray tracing
  statement {
    effect = "Allow"
    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords"
    ]

    resources = [
      "*",
    ]
  }
}

resource "aws_iam_policy" "function_logging_policy" {
  name        = "${var.domain_name}-RegistryLambdaCWLoggingPolicy"
  description = "Policy for the registry lambda to use cloudwatch logging"
  policy      = data.aws_iam_policy_document.allow_lambda_logging.json
}

resource "aws_iam_role_policy_attachment" "lambda_logging_policy_attachment" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.function_logging_policy.arn
}

data "aws_iam_policy_document" "dynamodb_policy" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:DescribeTable",
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:GetItem",
      "dynamodb:BatchGetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
      "dynamodb:BatchWriteItem"
    ]

    resources = [
      aws_dynamodb_table.provider_versions.arn,
      aws_dynamodb_table.module_versions.arn,
    ]
  }
}

resource "aws_iam_policy" "lambda_dynamo_policy" {
  name        = "${var.domain_name}-RegistryLambdaDynamoPolicy"
  description = "Policy for lambda to Read and Write to the provider versions DynamoDB table"
  policy      = data.aws_iam_policy_document.dynamodb_policy.json
}

resource "aws_iam_role_policy_attachment" "lambda_dynamo_policy_attachment" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.lambda_dynamo_policy.arn
}

// allow the api_function lambda to invoke the populate_provider_versions_function lambda
data "aws_iam_policy_document" "populate_provider_versions_policy" {
  statement {
    effect = "Allow"
    actions = [
      "lambda:InvokeFunction"
    ]

    resources = [
      aws_lambda_function.populate_provider_versions_function.arn,
      aws_lambda_function.populate_module_versions_function.arn
    ]
  }
}

resource "aws_iam_policy" "lambda_populate_provider_versions_policy" {
  name        = "${var.domain_name}-RegistryLambdaPopulateProviderVersionsPolicy"
  description = "Policy for the registry lambda to invoke the populate provider versions lambda"
  policy      = data.aws_iam_policy_document.populate_provider_versions_policy.json
}

resource "aws_iam_role_policy_attachment" "lambda_populate_provider_versions_policy_attachment" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.lambda_populate_provider_versions_policy.arn
}


