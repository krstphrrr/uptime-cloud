resource "aws_ecs_service" "uptime_monitor" {
  name            = "uptime-monitor-service"
  cluster         = aws_ecs_cluster.uptime.id
  launch_type     = "FARGATE"
  task_definition = aws_ecs_task_definition.uptime_monitor.arn
  desired_count   = 1

  network_configuration {
    subnets          = [aws_subnet.public_a.id]
    security_groups  = [aws_security_group.ecs_sg.id]
    assign_public_ip = true
  }

  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
}