variable "create" {
  description = "Whether to create EKS resources"
  type        = bool
}

variable "create_node_group" {
  description = "Whether to create EKS managed node group"
  type        = bool
  default     = true
}

variable "project" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "kubernetes_version" {
  description = "EKS Kubernetes version"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "cluster_subnet_ids" {
  description = "Subnet IDs for EKS control plane"
  type        = list(string)
}

variable "node_subnet_ids" {
  description = "Subnet IDs for EKS worker nodes"
  type        = list(string)
}

variable "node_group_instance_types" {
  description = "Node instance types"
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

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
