variable "environment" {
  type = string
}

variable "cluster_id" {
  type = string
}

variable "engine_version" {
  type = string
}

variable "node_type" {
  type = string
}

variable "num_cache_nodes" {
  type = number
}

variable "vpc_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "create_redis" {
  type = bool
}

variable "tags" {
  type = map(string)
}

resource "aws_elasticache_subnet_group" "main" {
  count      = var.create_redis ? 1 : 0
  name       = "redis-${var.environment}"
  subnet_ids = var.subnet_ids

  tags = var.tags
}

resource "aws_security_group" "redis" {
  count  = var.create_redis ? 1 : 0
  vpc_id = var.vpc_id
  name   = "redis-${var.environment}"

  ingress {
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  tags = var.tags
}

resource "aws_elasticache_cluster" "main" {
  count             = var.create_redis ? 1 : 0
  cluster_id        = var.cluster_id
  engine            = "redis"
  engine_version    = var.engine_version
  node_type         = var.node_type
  num_cache_nodes   = var.num_cache_nodes
  parameter_group_name = "default.redis${substr(var.engine_version, 0, 3)}"
  subnet_group_name = aws_elasticache_subnet_group.main[0].name
  security_group_ids = [aws_security_group.redis[0].id]

  tags = var.tags
}

output "redis_endpoint" {
  value = try(aws_elasticache_cluster.main[0].cache_nodes[0].address, "")
}
