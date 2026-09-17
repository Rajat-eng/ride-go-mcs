terraform {
  required_version = ">= 1.6.0"
}

resource "aws_security_group" "rabbitmq" {
  count = var.create ? 1 : 0

  name        = "${var.name_prefix}-rabbitmq-sg"
  description = "Security group for RabbitMQ"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 5672
    to_port     = 5672
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  ingress {
    from_port   = 15672
    to_port     = 15672
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

resource "aws_mq_broker" "this" {
  count = var.create ? 1 : 0

  broker_name                = "${var.name_prefix}-${var.broker_name}"
  engine_type                = "RabbitMQ"
  engine_version             = var.engine_version
  host_instance_type         = var.host_instance_type
  deployment_mode            = "SINGLE_INSTANCE"
  auto_minor_version_upgrade = true
  publicly_accessible        = false
  subnet_ids                 = [var.private_subnet_ids[0]]
  security_groups            = [aws_security_group.rabbitmq[0].id]

  user {
    username = var.username
    password = var.password
  }

  logs {
    general = true
  }

  tags = var.tags
}
