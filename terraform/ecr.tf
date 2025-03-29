resource "aws_ecr_repository" "uptime_monitor" {
  name = "uptime-monitor"

  image_scanning_configuration {
    scan_on_push = true
  }

  lifecycle {
    prevent_destroy = false
  }
}

output "ecr_repository_url" {
  description = "ECR repository URL for pushing uptime-monitor Docker images"
  value       = aws_ecr_repository.uptime_monitor.repository_url
}
