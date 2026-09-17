terraform {
  required_version = ">= 1.6.0"
}

resource "aws_elasticache_subnet_group" "this" {
  count = var.create ? 1 : 0

  name       = "${var.name_prefix}-redis-subnets"
  subnet_ids = var.private_subnet_ids
}

resource "aws_security_group" "redis" {
  count = var.create ? 1 : 0

  name        = "${var.name_prefix}-redis-sg"
  description = "Security group for Redis"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = var.port
    to_port     = var.port
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

resource "aws_elasticache_cluster" "this" {
  count = var.create ? 1 : 0

  cluster_id           = "${var.name_prefix}-${var.cluster_id}"
  engine               = "redis"
  node_type            = var.node_type
  num_cache_nodes      = var.num_cache_nodes
  parameter_group_name = var.parameter_group_name
  port                 = var.port
  subnet_group_name    = aws_elasticache_subnet_group.this[0].name
  security_group_ids   = [aws_security_group.redis[0].id]

  tags = var.tags
}
