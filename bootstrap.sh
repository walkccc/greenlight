source .envrc
make db/createdb dbname=greenlight
make db/create_role role=greenlight password=password
make db/create_extension dbname=greenlight extension=citext
make db/alter_database_owner dbname=greenlight username=greenlight
make db/migrate/up dsn=$GREENLIGHT_DB_DSN
