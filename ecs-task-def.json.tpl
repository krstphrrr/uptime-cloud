{
  "family": "uptime-monitor-task",
  "networkMode": "awsvpc",
  "executionRoleArn": "arn:aws:iam::969288771269:role/ecsTaskExecutionRole",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "uptime-monitor",
      "image": "969288771269.dkr.ecr.us-east-1.amazonaws.com/uptime-monitor:__VERSION__",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "hostPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "SMTP_HOST",
          "value": "smtp.gmail.com"
        },
        {
          "name": "SMTP_PORT",
          "value": "587"
        }
      ],
      "secrets": [
        {
          "name": "SMTP_USERNAME",
          "valueFrom": "__SMTP_USERNAME_ARN__"
        },
        {
          "name": "SMTP_PASSWORD",
          "valueFrom": "__SMTP_PASSWORD_ARN__"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/uptime-monitor",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
