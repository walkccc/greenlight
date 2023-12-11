include .envrc

# Global constants
POSTGRES_USER=root
POSTGRES_PASSWORD=password

# ============================================================================ #
# HELPERS
# ============================================================================ #

## help: Print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ============================================================================ #
# DEVELOPMENT
# ============================================================================ #

## run/api: Run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN}

## postgres: Run postgres by Docker
.PHONY: postgres
postgres:
	docker run --name postgres \
			-p 127.0.0.1:5432:5432/tcp \
			-e POSTGRES_USER=${POSTGRES_USER} \
			-e POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
			-d postgres:15.2-alpine

## db/createdb dbname=$1: Create a db
.PHONY: db/createdb
db/createdb:
	docker exec -it postgres createdb ${dbname}

## db/dropdb dbname=$1: Drop a db
.PHONY: db/dropdb
db/dropdb:
	docker exec -it postgres dropdb ${dbname}

## db/create_role role=$1 password=$2: Create a role
.PHONY: db/create_role
db/create_role:
	docker exec -it postgres \
			psql -U ${POSTGRES_USER} -c \
			"CREATE ROLE ${role} WITH LOGIN PASSWORD '${password}';"

## db/drop_role role=$1: Drop a role
.PHONY: db/drop_role
db/drop_role:
	docker exec -it postgres \
			psql -U ${POSTGRES_USER} -c \
			"DROP ROLE ${role};"

## db/create_extension dbname=$1 extension=$2: Create an extension in a db
.PHONY: db/create_extension
db/create_extension:
	docker exec -it postgres \
			psql -U ${POSTGRES_USER} -d ${dbname} -c \
			"CREATE EXTENSION IF NOT EXISTS ${extension};"

## db/psql dbname=$1 username=$2: Connect to a db using psql
.PHONY: db/psql
db/psql:
	docker exec -it postgres \
			psql --host=localhost --dbname=${dbname} --username=${username}

## db/alter_database_owner dbname=$1 username=$2: Alter the database owner
.PHONY: db/alter_database_owner
db/alter_database_owner:
	docker exec -it postgres \
			psql -U ${POSTGRES_USER} -c \
			"ALTER DATABASE ${dbname} OWNER TO ${username}"

## db/migrate/new name=$1: Create a new db migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrate/up dsn=$1: Apply all up db migrations
.PHONY: db/migrate/up
db/migrate/up:
	@echo 'Running up migrations...'
	migrate -path migrations -database=${GREENLIGHT_DB_DSN} -verbose up

## db/migrate/down dsn=$1: Apply all down db migrations
.PHONY: db/migrate/down
db/migrate/down:
	migrate -path migrations -database=${GREENLIGHT_DB_DSN} -verbose down

# ============================================================================ #
# QUALITY CONTROL
# ============================================================================ #

## audit: Tidy dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: Tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor
