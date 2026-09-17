output "redis_cluster_id" {
  description = "Redis cluster id"
  value       = try(aws_elasticache_cluster.this[0].id, null)
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = try(aws_elasticache_cluster.this[0].cache_nodes[0].address, null)
}

output "redis_port" {
  description = "Redis port"
  value       = try(aws_elasticache_cluster.this[0].cache_nodes[0].port, null)
}
