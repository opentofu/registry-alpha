terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    archive = {
      source = "hashicorp/archive"
    }
    null = {
      source = "hashicorp/null"
    }
  }

  required_version = "~> 1.5.0"
}

provider "aws" {
  region  = var.region

  default_tags {
    tags = {
      Project = "registry"
    }
  }
}