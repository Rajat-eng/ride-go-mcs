output "db_instance_id" {
  description = "RDS instance id"
  value       = try(aws_db_instance.this[0].id, null)
}

output "db_endpoint" {
  description = "RDS endpoint"
  value       = try(aws_db_instance.this[0].address, null)
}

output "db_port" {
  description = "RDS port"
  value       = try(aws_db_instance.this[0].port, null)
}
