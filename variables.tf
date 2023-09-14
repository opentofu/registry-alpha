variable "region" {
    default = "eu-west-1"
}

variable "github_api_token" {
    type = string
    sensitive = true
}

variable "route53_zone_id" {
    type = string
}

variable "domain_name" {
    type = string
}
