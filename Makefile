DATABASE_URL := "postgres://postgres:postgres@localhost:54321/premium_caste?sslmode=disable" 

run:
	go run cmd/premium_caste/main.go --config=./config/config.yaml

run-dev:
	docker-compose -f docker-compose.dev.yaml up --build

migration-up:
	goose --dir=./migrations postgres ${DATABASE_URL} up

migration-down:
	goose --dir=./migrations postgres ${DATABASE_URL} down

