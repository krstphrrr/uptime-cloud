resource "aws_ecr_repository" "uptime_monitor" {
  name = "uptime-monitor"

  image_scanning_configuration {
    scan_on_push = true
  }

  lifecycle {
    prevent_destroy = true
  }

  tags = {
    Name      = "uptime-monitor-ecr-repo"
    Project   = "uptime-monitor"
    ManagedBy = "Terraform"
  }
}

output "ecr_repository_url" {
  description = "ECR repository URL for pushing uptime-monitor Docker images"
  value       = aws_ecr_repository.uptime_monitor.repository_url
}
