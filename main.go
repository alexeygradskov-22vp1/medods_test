package main

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"medods/internal/api"
	"medods/internal/client/external"
	uow "medods/internal/repository/uow"
	"medods/internal/service/token"
	"medods/internal/service/user"
	"net/http"
	"os"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"),
		os.Getenv("PG_PORT"),
		os.Getenv("PG_USER"),
		os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_DB"),
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	externalServiceBaseUrl := os.Getenv("EXTERNAL_SERVER_HOST")
	if externalServiceBaseUrl == "" {
		panic("Missing EXTERNAL_SERVER_HOST")
	}
	unit := uow.NewUnitOfWork(db)
	httpClient := &http.Client{}
	client := external.NewClient(httpClient, externalServiceBaseUrl)
	ts := token.NewTService(unit, client)
	us := user.NewUService(unit, ts)
	api.StartHttpServer(ts, us)
}
