resource "null_resource" "api_function_binary" {
  provisioner "local-exec" {
    command     = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-trimpath go build -mod=readonly -ldflags='-s -w' -o ../${local.binary_name} ./lambda/api"
    working_dir = "./src"
  }

  triggers = {
    always_run = timestamp()
  }
}

data "archive_file" "api_function_archive" {
  depends_on = [null_resource.api_function_binary]

  type        = "zip"
  source_file = local.binary_name
  output_path = local.archive_path
}

// create the lambda function from zip file
resource "aws_lambda_function" "api_function" {
  function_name = "${replace(var.domain_name, ".", "-")}-registry-handler"
  description   = "A basic lambda to handle registry api events"
  role          = aws_iam_role.lambda.arn
  handler       = local.binary_name
  memory_size   = 128
  timeout       = 60

  filename         = local.archive_path
  source_code_hash = data.archive_file.api_function_archive.output_base64sha256

  runtime = "go1.x"

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      GITHUB_TOKEN_SECRET_ASM_NAME = aws_secretsmanager_secret.github_api_token.name
      PROVIDER_NAMESPACE_REDIRECTS = jsonencode(var.provider_namespace_redirects)
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

resource "aws_cloudwatch_log_group" "log_group" {
  name              = "/aws/lambda/${aws_lambda_function.api_function.function_name}"
  retention_in_days = 7
}
