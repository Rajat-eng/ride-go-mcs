output "cluster_name" {
  description = "EKS cluster name"
  value       = try(aws_eks_cluster.this[0].name, null)
}

output "cluster_endpoint" {
  description = "EKS API server endpoint"
  value       = try(aws_eks_cluster.this[0].endpoint, null)
}

output "cluster_security_group_id" {
  description = "EKS cluster security group id"
  value       = try(aws_security_group.eks_cluster[0].id, null)
}
