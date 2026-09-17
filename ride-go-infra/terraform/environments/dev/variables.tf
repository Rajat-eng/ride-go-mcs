variable "environment" {
  description = "Environment name"
  type        = string
}

variable "region" {
  description = "Cloud region"
  type        = string
}

variable "project" {
  description = "Project name used in resource naming"
  type        = string
}

variable "create_vpc" {
  description = "Whether to create VPC resources"
  type        = bool
}

variable "create_eks" {
  description = "Whether to create EKS cluster"
  type        = bool
}

variable "create_eks_node_group" {
  description = "Whether to create EKS managed node group"
  type        = bool
}

variable "create_rds" {
  description = "Whether to create RDS"
  type        = bool
}

variable "create_redis" {
  description = "Whether to create Redis"
  type        = bool
}

variable "create_rabbitmq" {
  description = "Whether to create RabbitMQ"
  type        = bool
}

variable "vpc_cidr" {
  description = "VPC CIDR"
  type        = string
}

variable "azs" {
  description = "Availability zones"
  type        = list(string)
}

variable "public_subnet_cidrs" {
  description = "Public subnet CIDRs"
  type        = list(string)
}

variable "private_subnet_cidrs" {
  description = "Private subnet CIDRs"
  type        = list(string)
}

variable "kubernetes_version" {
  description = "EKS kubernetes version"
  type        = string
}

variable "node_group_instance_types" {
  description = "Node group instance types"
  type        = list(string)
}

variable "node_desired_size" {
  description = "Desired node count"
  type        = number
}

variable "node_min_size" {
  description = "Minimum node count"
  type        = number
}

variable "node_max_size" {
  description = "Maximum node count"
  type        = number
}

variable "rds_db_name" {
  description = "RDS database name"
  type        = string
}

variable "rds_username" {
  description = "RDS username"
  type        = string
}

variable "rds_password" {
  description = "RDS password"
  type        = string
  sensitive   = true
}

variable "rds_instance_class" {
  description = "RDS instance class"
  type        = string
}

variable "rds_allocated_storage" {
  description = "RDS initial storage in GiB"
  type        = number
}

variable "redis_node_type" {
  description = "Redis node type"
  type        = string
}

variable "redis_num_cache_nodes" {
  description = "Redis node count"
  type        = number
}

variable "rabbitmq_host_instance_type" {
  description = "RabbitMQ instance type"
  type        = string
}

variable "rabbitmq_username" {
  description = "RabbitMQ username"
  type        = string
}

variable "rabbitmq_password" {
  description = "RabbitMQ password"
  type        = string
  sensitive   = true
}
