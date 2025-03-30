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

2. First with the infra-bootstrap user with temporary create permissions, navigate to the bootstrap directory:
```bash
aws configure --profile bootstrap
terraform init 
terraform plan 
terraform apply
```

3. Once IAM users are created, navigate to infrastructure directory and:
```bash
aws configure --profile github_deploy
aws configure --profile terraform
terraform init
terraform plan
terraform apply
```