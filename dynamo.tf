resource "aws_dynamodb_table" "provider_versions" {
  name         = "${var.domain_name}-provider-versions"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "provider"

  attribute {
    name = "provider"
    type = "S"
  }
}
resource "aws_dynamodb_table" "module_versions" {
  name         = "${var.domain_name}-module-versions"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "module"

  attribute {
    name = "module"
    type = "S"
  }
}