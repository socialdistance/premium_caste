DATABASE_URL := "postgres://postgres:postgres@localhost:54321/premium_caste?sslmode=disable" 

generate:
	protoc -I protoss/proto protoss/proto/auth_service/auth_service.proto --go_out=protoss/gen/go --go_opt=paths=source_relative --go-grpc_out=protoss/gen/go/ --go-grpc_opt=paths=source_relative

run_auth_service:
	go run ./auth_service/cmd/auth_service/main.go --config=./auth_service/config/local.yaml

auth_service_migrations:
	go run auth_service/cmd/migrator/main.go --storage-path=auth_service/storage/auth_service.db  --migrations-path=auth_service/migrations/

run_file_service:
	go run ./file_service/cmd/file_service/main.go --config=./file_service/config/local.yaml

run-race:
	go run ./file_service/cmd/file_service/main.go --config=./file_service/config/local.yaml

file_service_migration-up:
	goose --dir=./file_service/migrations postgres ${DATABASE_URL_FILE_SERVICE} up

file_service_migration-down:
	goose --dir=./file_service/migrations postgres ${DATABASE_URL_FILE_SERVICE} down

