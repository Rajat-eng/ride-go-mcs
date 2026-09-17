variable "environment" {
  type = string
}

variable "broker_name" {
  type = string
}

variable "engine_version" {
  type = string
}

variable "instance_type" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "create_broker" {
  type = bool
}

variable "admin_user" {
  type      = string
  sensitive = true
}

variable "admin_password" {
  type      = string
  sensitive = true
}

variable "tags" {
  type = map(string)
}

resource "aws_security_group" "rabbitmq" {
  count  = var.create_broker ? 1 : 0
  vpc_id = var.vpc_id
  name   = "mq-${var.environment}"

  ingress {
    from_port   = 5672
    to_port     = 5672
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    from_port   = 15672
    to_port     = 15672
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  tags = var.tags
}

resource "aws_mq_broker" "main" {
  count              = var.create_broker ? 1 : 0
  broker_name        = var.broker_name
  engine_type        = "RabbitMQ"
  engine_version     = var.engine_version
  host_instance_type = var.instance_type
  deployment_mode    = "SINGLE_INSTANCE"
  security_groups    = [aws_security_group.rabbitmq[0].id]
  subnet_ids         = var.subnet_ids

  user {
    username = var.admin_user
    password = var.admin_password
  }

  tags = var.tags
}

output "broker_amqp_endpoint" {
  value = try(aws_mq_broker.main[0].broker_node_instance_type, "")
}
