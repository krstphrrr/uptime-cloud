# terraform/iam.tf

# 1. GitHub Actions IAM user (for CI/CD)
resource "aws_iam_user" "github_deploy_user" {
  name = "github-deploy-user"
}

resource "aws_iam_access_key" "github_deploy_key" {
  user = aws_iam_user.github_deploy_user.name
}

resource "aws_iam_user_policy" "github_deploy_policy" {
  name = "github-deploy-policy"
  user = aws_iam_user.github_deploy_user.name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "ecr:*",
          "ecs:*",
          "secretsmanager:GetSecretValue",
          "logs:GetLogEvents",
          "logs:DescribeLogStreams"
        ],
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_policy" "github_deploy_passrole" {
  name = "GitHubDeployPassRole"
  description = "Allow GitHub Actions to pass ECS task execution role"
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = "iam:PassRole",
        Resource = "arn:aws:iam::969288771269:role/ecsTaskExecutionRole"
      }
    ]
  })
}

resource "aws_iam_user_policy_attachment" "github_deploy_passrole_attach" {
  user       = aws_iam_user.github_deploy_user.name
  policy_arn = aws_iam_policy.github_deploy_passrole.arn
}

# 2. Terraform local provisioning IAM user
resource "aws_iam_user" "terraform_provisioner" {
  name = "terraform-provisioner"
}

resource "aws_iam_access_key" "terraform_key" {
  user = aws_iam_user.terraform_provisioner.name
}

resource "aws_iam_user_policy" "terraform_policy" {
  name = "terraform-provisioner-policy"
  user = aws_iam_user.terraform_provisioner.name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "ecr:*",
          "ecs:*",
          "secretsmanager:*",
          "iam:*",
          "ec2:*",
          "logs:*"
        ],
        Resource = "*"
      }
    ]
  })
}

# Output credentials so you can set them up in GitHub/locally
output "github_deploy_user_credentials" {
  value = {
    access_key_id     = aws_iam_access_key.github_deploy_key.id
    secret_access_key = aws_iam_access_key.github_deploy_key.secret
  }
  sensitive = true
}

output "terraform_provisioner_credentials" {
  value = {
    access_key_id     = aws_iam_access_key.terraform_key.id
    secret_access_key = aws_iam_access_key.terraform_key.secret
  }
  sensitive = true
}
