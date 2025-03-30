variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "us-east-1"
}

data "aws_caller_identity" "current" {}

locals {
  account_id          = data.aws_caller_identity.current.account_id
  execution_role_arn  = "arn:aws:iam::${local.account_id}:role/ecsTaskExecutionRole"
}
