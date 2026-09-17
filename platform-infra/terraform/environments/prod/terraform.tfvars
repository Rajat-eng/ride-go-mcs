aws_region              = "ap-south-1"
environment             = "prod"
project_name            = "ride-go"
vpc_cidr                = "10.40.0.0/16"
azs                     = ["ap-south-1a", "ap-south-1b", "ap-south-1c"] // required availability zones for the VPC
public_subnet_cidrs     = ["10.40.1.0/24", "10.40.2.0/24", "10.40.3.0/24"] // required public subnets for the VPC
private_subnet_cidrs    = ["10.40.11.0/24", "10.40.12.0/24", "10.40.13.0/24"] // required private subnets for the VPC

# EKS Configuration (Production-grade)
kubernetes_version      = "1.30"
eks_instance_types      = ["t3.medium"]
eks_desired_size        = 2
eks_min_size            = 2
eks_max_size            = 4
create_eks_node_group   = true

# RDS Configuration (Production-grade: multi-AZ with encryption)
rds_instance_class      = "db.t4g.small"
rds_multi_az            = true
rds_username            = "admin"
rds_password            = "ChangeMe123!@#"
create_rds              = true

# Redis Configuration (Production-grade)
redis_node_type         = "cache.t3.small"
redis_num_nodes         = 1
create_redis            = true

# RabbitMQ Configuration (Production-grade)
rabbitmq_instance_type  = "mq.t3.small"
rabbitmq_admin_user     = "admin"
rabbitmq_admin_password = "ChangeMe123!@#"
create_rabbitmq         = true
