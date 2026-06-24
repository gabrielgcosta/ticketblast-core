.PHONY: sqlc migrate-create

# Generate Go code from SQL using sqlc
sqlc:
	sqlc generate -f db/sqlc.yaml

# Create a new migration file in db/migrations
# Usage: make migration name=create_users_table
migration:
	migrate create -ext sql -dir db/migrations -seq $(or $(name),create_users_table)
