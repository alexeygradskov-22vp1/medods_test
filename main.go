package main

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"medods/database"
	"medods/internal/api"
	"medods/internal/client/external"
	"medods/internal/repository/blacklist"
	token2 "medods/internal/repository/token"
	user2 "medods/internal/repository/user"
	"medods/internal/service/token"
	"medods/internal/service/user"
	"net/http"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	//database
	dbConf, err := database.NewConfig()
	if err != nil {
		panic(err)
	}
	db, err := database.NewDatabase(dbConf)
	if err != nil {
		panic(err)
	}

	//repositories
	uRepo := user2.NewRepository(db)
	tRepo := token2.NewRepository(db)
	blRepo := blacklist.NewRepository(db)

	//http client
	httpClient := &http.Client{}
	clientConf, err := external.NewConfig(httpClient)

	if err != nil {
		panic(err)
	}

	client := external.NewClient(clientConf)

	//services
	ts := token.NewTService(client, blRepo, tRepo)
	us := user.NewUService(uRepo, ts)

	api.StartHttpServer(ts, us)
}
