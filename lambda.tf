resource "null_resource" "api_function_binary" {
  provisioner "local-exec" {
    command     = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-trimpath go build -mod=readonly -tags lambda.norpc -ldflags='-s -w' -o ../api_function_bootstrap/bootstrap ./lambda/api"
    working_dir = "./src"
  }

  triggers = {
    always_run = timestamp()
  }
}

resource "null_resource" "populate_provider_versions_binary" {
  provisioner "local-exec" {
    command     = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-trimpath go build -mod=readonly -tags lambda.norpc -ldflags='-s -w' -o ../populate_provider_versions_bootstrap/bootstrap ./lambda/populate_provider_versions"
    working_dir = "./src"
  }

  triggers = {
    always_run = timestamp()
  }
}

data "archive_file" "api_function_archive" {
  depends_on = [null_resource.api_function_binary]

  type        = "zip"
  source_file = "./api_function_bootstrap/bootstrap"
  output_path = "api_bootstrap.zip"
}

data "archive_file" "populate_provider_versions_archive" {
  depends_on = [null_resource.populate_provider_versions_binary]

  type        = "zip"
  source_file = "./populate_provider_versions_bootstrap/bootstrap"
  output_path = "populate_provider_versions_bootstrap.zip"
}

// create the lambda function from zip file
resource "aws_lambda_function" "api_function" {
  function_name = "${replace(var.domain_name, ".", "-")}-registry-handler"
  description   = "A basic lambda to handle registry api events"
  role          = aws_iam_role.lambda.arn
  handler       = "registry-handler"
  memory_size   = 128
  timeout       = 60

  filename         = data.archive_file.api_function_archive.output_path
  source_code_hash = data.archive_file.api_function_archive.output_base64sha256

  runtime = "provided.al2"

  publish = true

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      GITHUB_TOKEN_SECRET_ASM_NAME = aws_secretsmanager_secret.github_api_token.name
      PROVIDER_NAMESPACE_REDIRECTS = jsonencode(var.provider_namespace_redirects)
      PROVIDER_VERSIONS_TABLE_NAME = aws_dynamodb_table.provider_versions.name
      POPULATE_PROVIDER_VERSIONS_FUNCTION_NAME = aws_lambda_function.populate_provider_versions_function.function_name
    }
  }
}

// ensure we have provisioned concurrency for the lambda function
resource "aws_lambda_provisioned_concurrency_config" "api_function" {
  function_name = aws_lambda_function.api_function.function_name
  provisioned_concurrent_executions = 1
  qualifier = aws_lambda_function.api_function.version
}

// create the lambda function from zip file
resource "aws_lambda_function" "populate_provider_versions_function" {
  function_name = "${replace(var.domain_name, ".", "-")}-populate-provider-versions"
  description   = "A basic lambda to handle populating provider versions in dynamodb"
  role          = aws_iam_role.lambda.arn
  handler       = "populate-provider-versions"
  memory_size   = 128
  timeout       = 60

  filename         = data.archive_file.populate_provider_versions_archive.output_path
  source_code_hash = data.archive_file.api_function_archive.output_base64sha256

  runtime = "provided.al2"

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      PROVIDER_VERSIONS_TABLE_NAME = aws_dynamodb_table.provider_versions.name
      GITHUB_TOKEN_SECRET_ASM_NAME = aws_secretsmanager_secret.github_api_token.name
    }
  }
}

resource "aws_lambda_permission" "api_gateway_invoke_lambda_permission" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.api_function.function_name}"
  principal     = "apigateway.amazonaws.com"

  # The /*/* portion grants access from any method on any resource
  # within the API Gateway "REST API".
  source_arn = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}

resource "aws_cloudwatch_log_group" "api_function_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.api_function.function_name}"
  retention_in_days = 7
}

resource "aws_cloudwatch_log_group" "populate_provider_versions_function_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.populate_provider_versions_function.function_name}"
  retention_in_days = 7
}