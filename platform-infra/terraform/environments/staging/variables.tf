variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-south-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "ride-go"
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
}

variable "azs" {
  description = "Availability zones"
  type        = list(string)
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets"
  type        = list(string)
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private subnets"
  type        = list(string)
}

variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.30"
}

variable "eks_instance_types" {
  description = "EC2 instance types for EKS node group"
  type        = list(string)
}

variable "eks_desired_size" {
  description = "Desired number of EKS nodes"
  type        = number
}

variable "eks_min_size" {
  description = "Minimum number of EKS nodes"
  type        = number
}

variable "eks_max_size" {
  description = "Maximum number of EKS nodes"
  type        = number
}

variable "create_eks_node_group" {
  description = "Create EKS node group"
  type        = bool
  default     = false
}

variable "rds_allocated_storage" {
  description = "Initial RDS storage in GB"
  type        = number
  default     = 20
}

variable "rds_max_allocated_storage" {
  description = "Maximum RDS autoscaling storage in GB"
  type        = number
  default     = 100
}

variable "rds_instance_class" {
  description = "RDS instance type"
  type        = string
}

variable "rds_multi_az" {
  description = "Enable RDS multi-AZ"
  type        = bool
  default     = false
}

variable "rds_username" {
  description = "RDS master username"
  type        = string
  sensitive   = true
}

variable "rds_password" {
  description = "RDS master password"
  type        = string
  sensitive   = true
}

variable "create_rds" {
  description = "Create RDS database"
  type        = bool
  default     = false
}

variable "redis_engine_version" {
  description = "Redis engine version"
  type        = string
  default     = "7.0"
}

variable "redis_node_type" {
  description = "Redis node type"
  type        = string
}

variable "redis_num_nodes" {
  description = "Number of Redis nodes"
  type        = number
  default     = 1
}

variable "create_redis" {
  description = "Create Redis cluster"
  type        = bool
  default     = false
}

variable "rabbitmq_engine_version" {
  description = "RabbitMQ engine version"
  type        = string
  default     = "3.12"
}

variable "rabbitmq_instance_type" {
  description = "RabbitMQ instance type"
  type        = string
}

variable "rabbitmq_admin_user" {
  description = "RabbitMQ admin username"
  type        = string
  sensitive   = true
}

variable "rabbitmq_admin_password" {
  description = "RabbitMQ admin password"
  type        = string
  sensitive   = true
}

variable "create_rabbitmq" {
  description = "Create RabbitMQ broker"
  type        = bool
  default     = false
}
