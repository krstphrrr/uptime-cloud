# ---------------------
# VPC + Subnet + Internet Gateway
# ---------------------
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "uptime-monitor-vpc"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "uptime-monitor-igw"
  }
}

resource "aws_subnet" "public_a" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "us-east-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "uptime-monitor-subnet-a"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "uptime-monitor-route-table"
  }
}

resource "aws_route_table_association" "public_a" {
  subnet_id      = aws_subnet.public_a.id
  route_table_id = aws_route_table.public.id
}

# ---------------------
# Security Group for Fargate (port 8080 open)
# ---------------------
resource "aws_security_group" "ecs_sg" {
  name        = "uptime-monitor-sg"
  description = "Allow inbound access to port 8080"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # Prometheus will be external
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "uptime-monitor-sg"
  }
}

# ---------------------
# ECS Cluster
# ---------------------
resource "aws_ecs_cluster" "uptime" {
  name = "uptime-monitor-cluster"

  tags = {
    Name = "uptime-monitor-cluster"
  }
}
