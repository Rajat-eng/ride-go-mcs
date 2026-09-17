aws_region              = "ap-south-1"
environment             = "dev"
project_name            = "ride-go"
vpc_cidr                = "10.20.0.0/16"
azs                     = ["ap-south-1a", "ap-south-1b"]
public_subnet_cidrs     = ["10.20.1.0/24", "10.20.2.0/24"]
private_subnet_cidrs    = ["10.20.11.0/24", "10.20.12.0/24"]

# EKS Configuration (disabled by default for cost optimization)
kubernetes_version      = "1.30"
eks_instance_types      = ["t3.small"]
eks_desired_size        = 1
eks_min_size            = 1
eks_max_size            = 3
create_eks_node_group   = true

# RDS Configuration (disabled by default for cost optimization)
rds_instance_class      = "db.t3.micro"
rds_multi_az            = false
rds_username            = "admin"
rds_password            = "ChangeMe123!@#"
create_rds              = false

# Redis Configuration (disabled by default for cost optimization)
redis_node_type         = "cache.t3.micro"
redis_num_nodes         = 1
create_redis            = false

# RabbitMQ Configuration (disabled by default for cost optimization)
rabbitmq_instance_type  = "mq.t3.micro"
rabbitmq_admin_user     = "admin"
rabbitmq_admin_password = "ChangeMe123!@#"
create_rabbitmq         = false
