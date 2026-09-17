variable "create" {
  description = "Whether to create RabbitMQ"
  type        = bool
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "broker_name" {
  description = "RabbitMQ broker name suffix"
  type        = string
}

variable "engine_version" {
  description = "RabbitMQ engine version"
  type        = string
  default     = "3.13"
}

variable "host_instance_type" {
  description = "Broker instance type"
  type        = string
}

variable "username" {
  description = "Broker username"
  type        = string
}

variable "password" {
  description = "Broker password"
  type        = string
  sensitive   = true
}

variable "private_subnet_ids" {
  description = "Private subnet IDs"
  type        = list(string)
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "allowed_cidr_blocks" {
  description = "CIDRs allowed to access RabbitMQ"
  type        = list(string)
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
