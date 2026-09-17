terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = var.environment
      Project     = var.project_name
      ManagedBy   = "Terraform"
      CreatedAt   = timestamp()
    }
  }
}

locals {
  tags = {
    Environment = var.environment
    Project     = var.project_name
  }
}

module "vpc" {
  source = "../modules/vpc"

  environment  = var.environment
  cidr_block   = var.vpc_cidr
  azs          = var.azs
  public_cidrs  = var.public_subnet_cidrs
  private_cidrs = var.private_subnet_cidrs
  tags         = local.tags
}

module "eks" {
  source = "../modules/eks"

  environment         = var.environment
  cluster_name        = "${var.project_name}-${var.environment}"
  kubernetes_version  = var.kubernetes_version
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_subnet_ids
  create_node_group   = var.create_eks_node_group
  node_group_name     = "${var.project_name}-${var.environment}-ng"
  instance_types      = var.eks_instance_types
  desired_size        = var.eks_desired_size
  min_size            = var.eks_min_size
  max_size            = var.eks_max_size
  tags                = local.tags
}

module "rds" {
  source = "../modules/rds"

  environment         = var.environment
  identifier          = "${var.project_name}-${var.environment}-db"
  allocated_storage   = var.rds_allocated_storage
  max_allocated_storage = var.rds_max_allocated_storage
  instance_class      = var.rds_instance_class
  multi_az            = var.rds_multi_az
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_subnet_ids
  create_database     = var.create_rds
  db_username         = var.rds_username
  db_password         = var.rds_password
  tags                = local.tags
}

module "redis" {
  source = "../modules/redis"

  environment         = var.environment
  cluster_id          = "${var.project_name}-${var.environment}-redis"
  engine_version      = var.redis_engine_version
  node_type           = var.redis_node_type
  num_cache_nodes     = var.redis_num_nodes
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_subnet_ids
  create_redis        = var.create_redis
  tags                = local.tags
}

module "rabbitmq" {
  source = "../modules/rabbitmq"

  environment         = var.environment
  broker_name         = "${var.project_name}-${var.environment}-mq"
  engine_version      = var.rabbitmq_engine_version
  instance_type       = var.rabbitmq_instance_type
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = [module.vpc.private_subnet_ids[0]]
  create_broker       = var.create_rabbitmq
  admin_user          = var.rabbitmq_admin_user
  admin_password      = var.rabbitmq_admin_password
  tags                = local.tags
}
