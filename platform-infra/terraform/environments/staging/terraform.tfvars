aws_region              = "ap-south-1"
environment             = "staging"
project_name            = "ride-go"
vpc_cidr                = "10.30.0.0/16"
azs                     = ["ap-south-1a", "ap-south-1b"]
public_subnet_cidrs     = ["10.30.1.0/24", "10.30.2.0/24"]
private_subnet_cidrs    = ["10.30.11.0/24", "10.30.12.0/24"]

# EKS Configuration
kubernetes_version      = "1.30"
eks_instance_types      = ["t3.medium"]
eks_desired_size        = 1
eks_min_size            = 1
eks_max_size            = 3
create_eks_node_group   = true

# RDS Configuration
rds_instance_class      = "db.t3.small"
rds_multi_az            = false
rds_username            = "admin"
rds_password            = "ChangeMe123!@#"
create_rds              = true

# Redis Configuration
redis_node_type         = "cache.t3.small"
redis_num_nodes         = 1
create_redis            = true

# RabbitMQ Configuration
rabbitmq_instance_type  = "mq.t3.small"
rabbitmq_admin_user     = "admin"
rabbitmq_admin_password = "ChangeMe123!@#"
create_rabbitmq         = true
