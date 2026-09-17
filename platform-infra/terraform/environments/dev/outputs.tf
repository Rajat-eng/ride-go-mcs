output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = try(module.eks.cluster_name, "")
}

output "eks_cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = try(module.eks.cluster_endpoint, "")
}

output "rds_endpoint" {
  description = "RDS database endpoint"
  value       = try(module.rds.db_endpoint, "")
}

output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = try(module.redis.redis_endpoint, "")
}

output "rabbitmq_endpoint" {
  description = "RabbitMQ broker AMQP endpoint"
  value       = try(module.rabbitmq.broker_amqp_endpoint, "")
}
