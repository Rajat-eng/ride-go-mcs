variable "create" {
  description = "Whether to create Redis"
  type        = bool
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "cluster_id" {
  description = "Redis cluster id suffix"
  type        = string
}

variable "node_type" {
  description = "Redis node type"
  type        = string
}

variable "num_cache_nodes" {
  description = "Number of cache nodes"
  type        = number
}

variable "parameter_group_name" {
  description = "Redis parameter group"
  type        = string
  default     = "default.redis7"
}

variable "port" {
  description = "Redis port"
  type        = number
  default     = 6379
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
  description = "CIDRs allowed to access Redis"
  type        = list(string)
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
