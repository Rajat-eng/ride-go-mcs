output "rabbitmq_broker_id" {
  description = "RabbitMQ broker id"
  value       = try(aws_mq_broker.this[0].id, null)
}

output "rabbitmq_amqp_endpoint" {
  description = "RabbitMQ AMQP endpoint"
  value       = try(aws_mq_broker.this[0].instances[0].endpoints[0], null)
}
