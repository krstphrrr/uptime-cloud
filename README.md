## uptime cloud

1. create `infra-bootstrap` aws user with the following permission policy:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "*",
      "Resource": "*"
    }
  ]
}
```

2. Using the `infra-bootstrap` user, use the terraform plan in the bootstrap directory to create the limited IAM users that will set up everything else:
```bash
aws configure --profile bootstrap
terraform init 
terraform plan 
terraform apply

# deactivate access for the infra-bootstrap user after usage
```

3. Once IAM users are created, navigate to infrastructure directory and run terraform there:
```bash
# adds the two just created limited aws users that will set up infrastructure
aws configure --profile github_deploy
aws configure --profile terraform

# terraform will leverage these two users to prop up resources
terraform init
terraform plan
terraform apply
```

4. To destroy the resources:
```bash
# to destroy all resources:
terraform destroy
# in this scenario, I added the prevent_destroy: true on the ECR repository, and had to manually target each service.
terraform destroy -target=aws_ecs_cluster.uptime \    
                  -target=aws_ecs_service.uptime_monitor \                  
                  -target=aws_ecs_task_definition.uptime_monitor \                  
                  -target=aws_vpc.main \                  
                  -target=aws_subnet.public_a \                 
                  -target=aws_security_group.ecs_sg \                 
                  -target=aws_internet_gateway.main \                  
                  -target=aws_route_table.public \                 
                  -target=aws_route_table_association.public_a \                
                  -target=aws_cloudwatch_log_group.uptime_monitor \              
                  -target=aws_secretsmanager_secret.smtp_username \               
                  -target=aws_secretsmanager_secret.smtp_password \                  -target=aws_secretsmanager_secret_version.smtp_username \                  -target=aws_secretsmanager_secret_version.smtp_password \                 -target=aws_iam_role_policy_attachment.ecs_task_execution_policy \                 -target=aws_iam_policy_attachment.ecs_secrets_access

# it may be worth not destroying secrets as they are not destroyed immediately, but rather are 'scheduled for deletion' which can take 7-30 days

```