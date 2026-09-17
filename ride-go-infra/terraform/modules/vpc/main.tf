terraform {
  required_version = ">= 1.6.0"
}

resource "aws_vpc" "this" {
  count = var.create ? 1 : 0

  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge(var.tags, {
    Name = "${var.name_prefix}-vpc"
  })
}

resource "aws_internet_gateway" "this" {
  count = var.create && var.create_public_subnets ? 1 : 0

  vpc_id = aws_vpc.this[0].id

  tags = merge(var.tags, {
    Name = "${var.name_prefix}-igw"
  })
}

locals {
  public_subnet_map = {
    for idx, cidr in var.public_subnet_cidrs : idx => cidr
  }
  private_subnet_map = {
    for idx, cidr in var.private_subnet_cidrs : idx => cidr
  }
}

resource "aws_subnet" "public" {
  for_each = var.create && var.create_public_subnets ? local.public_subnet_map : {}

  vpc_id                  = aws_vpc.this[0].id
  cidr_block              = each.value
  availability_zone       = var.azs[tonumber(each.key)]
  map_public_ip_on_launch = true

  tags = merge(var.tags, {
    Name = "${var.name_prefix}-public-${each.key}"
  })
}

resource "aws_subnet" "private" {
  for_each = var.create ? local.private_subnet_map : {}

  vpc_id            = aws_vpc.this[0].id
  cidr_block        = each.value
  availability_zone = var.azs[tonumber(each.key)]

  tags = merge(var.tags, {
    Name = "${var.name_prefix}-private-${each.key}"
  })
}

resource "aws_route_table" "public" {
  count = var.create && var.create_public_subnets ? 1 : 0

  vpc_id = aws_vpc.this[0].id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this[0].id
  }

  tags = merge(var.tags, {
    Name = "${var.name_prefix}-public-rt"
  })
}

resource "aws_route_table_association" "public" {
  for_each = var.create && var.create_public_subnets ? aws_subnet.public : {}

  subnet_id      = each.value.id
  route_table_id = aws_route_table.public[0].id
}
