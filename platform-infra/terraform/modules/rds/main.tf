variable "environment" {
  type = string
}

variable "identifier" {
  type = string
}

variable "allocated_storage" {
  type = number
}

variable "max_allocated_storage" {
  type = number
}

variable "instance_class" {
  type = string
}

variable "multi_az" {
  type = bool
}

variable "vpc_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "create_database" {
  type = bool
}

variable "db_username" {
  type      = string
  sensitive = true
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "tags" {
  type = map(string)
}

resource "aws_db_subnet_group" "main" {
  count           = var.create_database ? 1 : 0
  name            = "rds-${var.environment}"
  subnet_ids      = var.subnet_ids
  tags            = var.tags
}

resource "aws_security_group" "rds" {
  count  = var.create_database ? 1 : 0
  vpc_id = var.vpc_id
  name   = "rds-${var.environment}"

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  tags = var.tags
}

resource "aws_db_instance" "main" {
  count                       = var.create_database ? 1 : 0
  identifier                  = var.identifier
  engine                      = "postgres"
  engine_version              = "15.3"
  instance_class              = var.instance_class
  allocated_storage           = var.allocated_storage
  max_allocated_storage       = var.max_allocated_storage
  username                    = var.db_username
  password                    = var.db_password
  multi_az                    = var.multi_az
  db_subnet_group_name        = aws_db_subnet_group.main[0].name
  vpc_security_group_ids      = [aws_security_group.rds[0].id]
  storage_encrypted           = true
  skip_final_snapshot         = true
  backup_retention_period     = var.multi_az ? 7 : 1

  tags = var.tags
}

output "db_endpoint" {
  value = try(aws_db_instance.main[0].endpoint, "")
}
