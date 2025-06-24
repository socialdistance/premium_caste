DATABASE_URL := "postgres://postgres:postgres@localhost:54321/premium_caste?sslmode=disable" 

generate:
	protoc -I protoss/proto protoss/proto/auth_service/auth_service.proto --go_out=protoss/gen/go --go_opt=paths=source_relative --go-grpc_out=protoss/gen/go/ --go-grpc_opt=paths=source_relative

run:
	go run cmd/premium_caste/main.go --config=./config/config.yaml

run-dev:
	docker-compose -f docker-compose.dev.yaml up --build

run-race:
	go run ./file_service/cmd/file_service/main.go --config=./file_service/config/local.yaml

migration-up:
	goose --dir=./migrations postgres ${DATABASE_URL} up

migration-down:
	goose --dir=./migrations postgres ${DATABASE_URL} down

