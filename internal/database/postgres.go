package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PgPool *pgxpool.Pool
var PgCtx = context.Background()

func InitPostgres() error {
	var err error
	PgPool, err = pgxpool.New(PgCtx, os.Getenv("DB_URL"))
	return err
}
