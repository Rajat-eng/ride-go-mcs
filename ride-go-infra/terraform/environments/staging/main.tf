terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region

  default_tags {
    tags = {
      project     = var.project
      environment = var.environment
      managed_by  = "terraform"
    }
  }
}

module "vpc" {
  source = "../../modules/vpc"

  create                = var.create_vpc
  name_prefix           = "${var.project}-${var.environment}"
  vpc_cidr              = var.vpc_cidr
  azs                   = var.azs
  public_subnet_cidrs   = var.public_subnet_cidrs
  private_subnet_cidrs  = var.private_subnet_cidrs
  create_public_subnets = true

  tags = local.tags
}

module "eks" {
  source = "../../modules/eks"

  create                     = var.create_eks
  create_node_group          = var.create_eks_node_group
  project                    = var.project
  environment                = var.environment
  kubernetes_version         = var.kubernetes_version
  vpc_id                     = module.vpc.vpc_id
  cluster_subnet_ids         = module.vpc.private_subnet_ids
  node_subnet_ids            = module.vpc.private_subnet_ids
  node_group_instance_types  = var.node_group_instance_types
  node_desired_size          = var.node_desired_size
  node_min_size              = var.node_min_size
  node_max_size              = var.node_max_size

  tags = local.tags
}

module "rds" {
  source = "../../modules/rds"

  create              = var.create_rds
  name_prefix         = "${var.project}-${var.environment}"
  identifier          = "postgres"
  db_name             = var.rds_db_name
  username            = var.rds_username
  password            = var.rds_password
  instance_class      = var.rds_instance_class
  allocated_storage   = var.rds_allocated_storage
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  allowed_cidr_blocks = [var.vpc_cidr]
  multi_az            = false

  tags = local.tags
}

module "redis" {
  source = "../../modules/redis"

  create              = var.create_redis
  name_prefix         = "${var.project}-${var.environment}"
  cluster_id          = "redis"
  node_type           = var.redis_node_type
  num_cache_nodes     = var.redis_num_cache_nodes
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  allowed_cidr_blocks = [var.vpc_cidr]

  tags = local.tags
}

module "rabbitmq" {
  source = "../../modules/rabbitmq"

  create              = var.create_rabbitmq
  name_prefix         = "${var.project}-${var.environment}"
  broker_name         = "rabbitmq"
  host_instance_type  = var.rabbitmq_host_instance_type
  username            = var.rabbitmq_username
  password            = var.rabbitmq_password
  private_subnet_ids  = module.vpc.private_subnet_ids
  vpc_id              = module.vpc.vpc_id
  allowed_cidr_blocks = [var.vpc_cidr]

  tags = local.tags
}

locals {
  tags = {
    project     = var.project
    environment = var.environment
  }
}
