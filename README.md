# Greenlight

<a href="https://golang.org/doc/go1.21"><img alt="Go 1.21" src="https://img.shields.io/badge/Go-1.21-blue?logo=go&color=5EC9E3"></a>

> This repository is based on
> ["Let's Go Further" by Alex Edwards](https://lets-go-further.alexedwards.net).
> The **Go**al is to provide a well-structured way to start a Go-backed backend
> infrastructure.

## Prerequesites

### Install Docker and PostgreSQL image

```bash
# Install Docker.
brew install docker

# Run Docker app so that we can access the `docker` command.

# Pull the PostgresSQL image.
docker pull postgres:15.5-alpine

# Check the downloaded image.
docker images
```

### Install `migrate`

```bash
# Install `migrate` command.
brew install golang-migrate

# Check the installed `migrate` command.
migrate --version
```

### Take a look at [`Makefile`](./Makefile) and [`bootstrap.sh`](./bootstrap.sh)

```bash
# See all the available commands.
make help
```

## Get Started

### Run a Docker container using the official PostgreSQL image

Creates and runs a Docker container with the name `postgres`, using the official
`postgres:15.5-alpine` Docker image. The container is started as a background
process (`-d` flag) and is mapped to port `5432` of the host machine
(`-p 127.0.0.1:5432:5432/tcp` flag), which is the default port for PostgreSQL.

The container is also configured with the environment variables `POSTGRES_USER`
and `POSTGRES_PASSWORD`, which set the default username and password for the
PostgreSQL database. In this case, the username is set to `root` and the
password is set to `password`.

```bash
docker run --name postgres \
  -p 127.0.0.1:5432:5432/tcp \
  -e POSTGRES_USER=root \
  -e POSTGRES_PASSWORD=password \
  -d postgres:15.5-alpine
```

```bash
# Interact with a PostgreSQL database running inside a Docker container named
# 'postgres'. Open an interactive terminal (`psql`) as the 'root' user for
# executing SQL commands.
docker exec -it postgres psql -U root

# Try the following query in the shell.
SELECT NOW();
```

### Create `.envrc`

```bash
touch .envrc
echo "export GREENLIGHT_DB_DSN=postgres://pengyuc:password@localhost:5432/greenlight?sslmode=disable" >> .envrc
```

### Run `bootstrap.sh` to create db, role, extension and migrate the db

```bash
bash bootstrap.sh
```

## Style Guide

While cases and spaces don't make a difference in the SQL engine, except for the
size of the raw file, I aim to stay consistent and enhance readability whenever
possible, just as in my other projects.

- Use `UpperCase` for table names, which might be slightly different from what's
  in the book.
- Use `lower_snake_case` for column names.
- Capitalize the keywords in Postgres.

## Visual Studio Code extensions

```bash
code --install-extension esbenp.prettier-vscode
code --install-extension foxundermoon.shell-format
code --install-extension golang.go
```

## Appendix

### Add new migration scripts

```bash
migrate create -ext sql -dir migrations -seq <new_script_file_name>
```
