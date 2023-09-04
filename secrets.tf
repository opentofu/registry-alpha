resource "aws_secretsmanager_secret" "github_api_token" {
  name = "github_api_token"
}

resource "aws_secretsmanager_secret_version" "github_api_token" {
  secret_id = aws_secretsmanager_secret.github_api_token.id
  secret_string = var.github_api_token
}