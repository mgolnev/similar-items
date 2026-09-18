package database

// RunMigrations накатывает схему. Движок работает только на Postgres,
// нативные PG-миграции — в migrations_postgres.go.
func RunMigrations(db *DB) error {
	return RunPostgresMigrations(db)
}
