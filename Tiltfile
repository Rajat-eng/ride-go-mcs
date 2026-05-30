load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###

# Uncomment to use secrets
k8s_yaml('./infra/development/k8s/secrets.yaml')
k8s_yaml('./infra/development/k8s/app-config.yaml')

### End of K8s Config ###

### RabbitMQ ###
k8s_yaml('./infra/development/k8s/rabbitmq-deployment.yaml')
k8s_resource('rabbitmq', port_forwards=['5672', '15672'], labels='tooling')
### End RabbitMQ ###

### Redis ###
k8s_yaml('./infra/development/k8s/redis-deployment.yaml')
k8s_resource('redis', port_forwards=6379, labels='tooling')
### End Redis ###

### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway'
if os.name == 'nt':
  gateway_compile_cmd = './infra/development/docker/api-gateway-build.bat'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")


docker_build_with_restart(
  'ride-sharing/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/api-gateway-deployment.yaml')
k8s_resource('api-gateway', port_forwards=8081,
             resource_deps=['api-gateway-compile','rabbitmq'], labels="services")
### End of API Gateway ###
### Trip Service ###


trip_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/trip-service ./services/trip-service/cmd/main.go'
if os.name == 'nt':
 trip_compile_cmd = './infra/development/docker/trip-build.bat'

local_resource(
  'trip-service-compile',
  trip_compile_cmd,
  deps=['./services/trip-service', './shared'], labels="compiles")

docker_build_with_restart(
  'ride-sharing/trip-service',
  '.',
  entrypoint=['/app/build/trip-service'],
  dockerfile='./infra/development/docker/trip-service.Dockerfile',
  only=[
    './build/trip-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/trip-service-deployment.yaml')
k8s_resource('trip-service', resource_deps=['trip-service-compile','rabbitmq'], labels="services")

## end of trip service ##

## login service

login_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/login-service ./services/login-service/cmd/main.go'
if os.name == 'nt':
    login_compile_cmd = './infra/development/docker/login-build.bat'  
local_resource(
  'login-service-compile',
  login_compile_cmd,
  deps=['./services/login-service', './shared'], labels="compiles")
docker_build_with_restart(
  'ride-sharing/login-service', # image name
  '.',
  entrypoint=['/app/build/login-service'], 
  dockerfile='./infra/development/docker/login-service.Dockerfile',
  only=[ # only list provides context for Docker build, not for live updates
    './build/login-service',
    './shared',
    './services/login-service/migrations', # Include migrations in the Docker image
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/login-service-deployment.yaml')
k8s_resource('login-service', resource_deps=['login-service-compile'], labels="services")

## End of login service ##

### Driver Service ###
driver_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/driver-service ./services/driver-service'
if os.name == 'nt':
 driver_compile_cmd = './infra/development/docker/driver-build.bat'

local_resource(
  'driver-service-compile',
  driver_compile_cmd,
  deps=['./services/driver-service', './shared'], labels="compiles")

docker_build_with_restart(
  'ride-sharing/driver-service',
  '.',
  entrypoint=['/app/build/driver-service'],
  dockerfile='./infra/development/docker/driver-service.Dockerfile',
  only=[
    './build/driver-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/driver-service-deployment.yaml')
k8s_resource('driver-service', resource_deps=['driver-service-compile','rabbitmq'], labels="services")

### End of Driver Service ###



### DLQ Worker ###

dlq_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/dlq-worker ./services/dlq-worker'
if os.name == 'nt':
  dlq_compile_cmd = './infra/development/docker/dlq-build.bat'

local_resource(
  'dlq-worker-compile',
  dlq_compile_cmd,
  deps=['./services/dlq-worker', './shared'],
  labels="compiles"
)

docker_build_with_restart(
  'ride-sharing/dlq-worker',
  '.',
  entrypoint=['/app/build/dlq-worker'],
  dockerfile='./infra/development/docker/dlq-worker.Dockerfile',
  only=[
    './build/dlq-worker',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/dlq-worker-deployment.yaml')
k8s_resource(
  'dlq-worker',
  resource_deps=['dlq-worker-compile', 'rabbitmq'],
  labels="jobs"
)

### End of DLQ Worker ###



### Web Frontend ###

docker_build(
  'ride-sharing/web',
  '.',
  dockerfile='./infra/development/docker/web.Dockerfile',
  build_args={
    'NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY': os.getenv('NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY', ''),
  },
)

k8s_yaml('./infra/development/k8s/web-deployment.yaml')
k8s_resource('web', port_forwards=3000, labels="frontend")

### End of Web Frontend ###

### Payment Service ###

payment_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/payment-service ./services/payment-service/cmd/main.go'
if os.name == 'nt':
  payment_compile_cmd = './infra/development/docker/payment-build.bat'

local_resource(
  'payment-service-compile',
  payment_compile_cmd,
  deps=['./services/payment-service', './shared'], labels="compiles")

docker_build_with_restart(
  'ride-sharing/payment-service',
  '.',
  entrypoint=['/app/build/payment-service'],
  dockerfile='./infra/development/docker/payment-service.Dockerfile',
  only=[
    './build/payment-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/payment-service-deployment.yaml')
k8s_resource('payment-service', resource_deps=['payment-service-compile', 'rabbitmq'], labels="services")

### End of Payment Service ###

### WS Gateway ###

ws_gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/ws-gateway ./services/ws-gateway'
if os.name == 'nt':
  ws_gateway_compile_cmd = './infra/development/docker/ws-gateway-build.bat'

local_resource(
  'ws-gateway-compile',
  ws_gateway_compile_cmd,
  deps=['./services/ws-gateway', './shared'], labels="compiles")

docker_build_with_restart(
  'ride-sharing/ws-gateway',
  '.',
  entrypoint=['/app/build/ws-gateway'],
  dockerfile='./infra/development/docker/ws-gateway.Dockerfile',
  only=[
    './build/ws-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/ws-gateway-deployment.yaml')
k8s_resource('ws-gateway', port_forwards=8082,
             resource_deps=['ws-gateway-compile', 'rabbitmq', 'redis'], labels="services")

### End of WS Gateway ###

### Chat Service ###

chat_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/chat-service ./services/chat-service/cmd/main.go'
if os.name == 'nt':
  chat_compile_cmd = './infra/development/docker/chat-build.bat'

local_resource(
  'chat-service-compile',
  chat_compile_cmd,
  deps=['./services/chat-service', './shared'], labels="compiles")

docker_build_with_restart(
  'ride-sharing/chat-service',
  '.',
  entrypoint=['/app/build/chat-service'],
  dockerfile='./infra/development/docker/chat-service.Dockerfile',
  only=[
    './build/chat-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/chat-service-deployment.yaml')
k8s_resource('chat-service', resource_deps=['chat-service-compile', 'rabbitmq'], labels="services")

### End of Chat Service ###

### Jaeger ###
k8s_yaml('./infra/development/k8s/jaeger.yaml')
k8s_resource(
  'jaeger',
  port_forwards=['16686:16686', '14268:14268'],
  resource_deps=[
    'rabbitmq',
    'redis',
    'postgres',
    'api-gateway',
    'trip-service',
    'login-service',
    'driver-service',
    'dlq-worker',
    'web',
    'payment-service',
    'ws-gateway',
    'chat-service',
  ],
  labels="tooling",
)
### End of Jaeger ###


###Postges ###
k8s_yaml('./infra/development/k8s/postgres-deployment.yaml')  
k8s_resource('postgres', port_forwards=5432, labels='tooling')
### End of Postgres ###