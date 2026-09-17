environment = "staging"
region      = "ap-south-1"
project     = "ride-go"

create_vpc            = true
create_eks            = true
create_eks_node_group = true
create_rds            = false
create_redis          = false
create_rabbitmq       = false

vpc_cidr             = "10.30.0.0/16"
azs                  = ["ap-south-1a", "ap-south-1b"]
public_subnet_cidrs  = ["10.30.1.0/24", "10.30.2.0/24"]
private_subnet_cidrs = ["10.30.11.0/24", "10.30.12.0/24"]

kubernetes_version        = "1.30"
node_group_instance_types = ["t3.medium"]
node_desired_size         = 1
node_min_size             = 1
node_max_size             = 3

rds_db_name           = "ridego"
rds_username          = "ridego_admin"
rds_password          = "replace-me-rds-password"
rds_instance_class    = "db.t4g.micro"
rds_allocated_storage = 20

redis_node_type       = "cache.t4g.micro"
redis_num_cache_nodes = 1

rabbitmq_host_instance_type = "mq.t3.micro"
rabbitmq_username           = "ridego_admin"
rabbitmq_password           = "replace-me-rabbitmq-password"
