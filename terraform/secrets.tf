variable "smtp_username" {
  description = "SMTP username"
  type        = string
  sensitive   = true
}

variable "smtp_password" {
  description = "SMTP password"
  type        = string
  sensitive   = true
}

resource "aws_secretsmanager_secret" "smtp_username" {
  name = "smtp_username"
}

resource "aws_secretsmanager_secret_version" "smtp_username" {
  secret_id     = aws_secretsmanager_secret.smtp_username.id
  secret_string = var.smtp_username
}

resource "aws_secretsmanager_secret" "smtp_password" {
  name = "smtp_password"
}

resource "aws_secretsmanager_secret_version" "smtp_password" {
  secret_id     = aws_secretsmanager_secret.smtp_password.id
  secret_string = var.smtp_password
}
