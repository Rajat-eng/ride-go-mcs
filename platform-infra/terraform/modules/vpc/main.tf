variable "environment" {
  type = string
}

variable "cidr_block" {
  type = string
}

variable "azs" {
  type = list(string)
}

variable "public_cidrs" {
  type = list(string)
}

variable "private_cidrs" {
  type = list(string)
}

variable "tags" {
  type = map(string)
}

resource "aws_vpc" "main" {
  cidr_block           = var.cidr_block
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(var.tags, { Name = "vpc-${var.environment}" })
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(var.tags, { Name = "igw-${var.environment}" })
}

resource "aws_eip" "nat" {
  domain = "vpc"
  count  = length(var.azs)

  tags = merge(var.tags, { Name = "eip-nat-${var.environment}-${var.azs[count.index]}" })

  depends_on = [aws_internet_gateway.main]
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_cidrs[count.index]
  availability_zone       = var.azs[count.index]
  count                   = length(var.azs)
  map_public_ip_on_launch = true

  tags = merge(var.tags, { Name = "subnet-public-${var.environment}-${var.azs[count.index]}" })
}

resource "aws_subnet" "private" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_cidrs[count.index]
  availability_zone = var.azs[count.index]
  count             = length(var.azs)

  tags = merge(var.tags, { Name = "subnet-private-${var.environment}-${var.azs[count.index]}" })
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  count         = length(var.azs)

  tags = merge(var.tags, { Name = "nat-${var.environment}-${var.azs[count.index]}" })

  depends_on = [aws_internet_gateway.main]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block      = "0.0.0.0/0"
    gateway_id      = aws_internet_gateway.main.id
  }

  tags = merge(var.tags, { Name = "rt-public-${var.environment}" })
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id
  count  = length(var.azs)

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main[count.index].id
  }

  tags = merge(var.tags, { Name = "rt-private-${var.environment}-${var.azs[count.index]}" })
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
  count          = length(var.azs)
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
  count          = length(var.azs)
}

output "vpc_id" {
  value = aws_vpc.main.id
}

output "public_subnet_ids" {
  value = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  value = aws_subnet.private[*].id
}
