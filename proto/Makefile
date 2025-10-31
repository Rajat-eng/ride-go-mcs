PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
GO_OUT := .

.PHONY: generate-proto
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \ # Specify the directory where your .proto files are located
		--go_out=$(GO_OUT) \ # Generate Go code for the messages 
		--go-grpc_out=$(GO_OUT) \ # Generate Go code for gRPC services 
		$(PROTO_SRC)


# 		protoc --proto_path=proto \
#        --java_out=src/main/java \
#        --grpc-java_out=src/main/java \
#        proto/trip.proto