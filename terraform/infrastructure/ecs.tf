# ---------------------
# CloudWatch Log Group
# ---------------------
resource "aws_cloudwatch_log_group" "uptime_monitor" {
  name              = "/ecs/uptime-monitor"
  retention_in_days = 7
  tags = {
    Name = "uptime-monitor-cloudwatch-log-group"
    Project = "uptime-monitor"
    ManagedBy = "Terraform"
  }
}

# ---------------------
# Task Execution Role (allows ECS to pull image, read secrets)
# ---------------------
resource "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsTaskExecutionRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_policy" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_policy_attachment" "ecs_secrets_access" {
  name       = "ecs_secrets_access"
  roles      = [aws_iam_role.ecs_task_execution_role.name]
  policy_arn = "arn:aws:iam::aws:policy/SecretsManagerReadWrite"
}

# ---------------------
# Task Definition
# ---------------------
resource "aws_ecs_task_definition" "uptime_monitor" {
  family                   = "uptime-monitor-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn

  container_definitions = jsonencode([
    {
      name      = "uptime-monitor"
      image     = "${aws_ecr_repository.uptime_monitor.repository_url}:latest"
      portMappings = [{
        containerPort = 8080
        hostPort      = 8080
        protocol      = "tcp"
      }]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.uptime_monitor.name
          awslogs-region        = "us-east-1"
          awslogs-stream-prefix = "ecs"
        }
      }
      environment = [
        {
          name  = "SMTP_HOST"
          value = "smtp.gmail.com"
        },
        {
          name  = "SMTP_PORT"
          value = "587"
        }
      ]
      secrets = [
        {
          name      = "SMTP_USERNAME"
          valueFrom = aws_secretsmanager_secret.smtp_username.arn
        },
        {
          name      = "SMTP_PASSWORD"
          valueFrom = aws_secretsmanager_secret.smtp_password.arn
        }
      ]
    }
  ])

  tags = {
    Name = "uptime-monitor-task"
    Project = "uptime-monitor"
    ManagedBy = "Terraform"
  }
}
