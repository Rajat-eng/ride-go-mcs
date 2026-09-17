terraform {
  required_version = ">= 1.6.0"
}

# Minimal stub for Kubernetes cluster (EKS/GKE/AKS) resources.
locals {
  module_name = "eks"
}
