resource "aws_dynamodb_table" "provider_versions" {
  name         = "provider-versions"
  billing_mode = "PAY_PER_REQUEST"

  hash_key     = "provider"

  attribute {
    name = "provider"
    type = "S"
  }
}