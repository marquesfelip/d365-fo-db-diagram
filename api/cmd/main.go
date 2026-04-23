package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/database"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/server"
)

func main() {

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("error loading .env file: ", err)
	}

	port := os.Getenv("PORT")

	pg_host := os.Getenv("DATABASE_HOST")
	pg_user := os.Getenv("POSTGRES_USER")
	pg_pw := os.Getenv("POSTGRES_PASSWORD")
	pg_db := os.Getenv("POSTGRES_DB")
	pg_port := os.Getenv("POSTGRES_PORT")

	if port == "" {
		log.Fatal("backend PORT is not defined")
	}

	if pg_host == "" || pg_user == "" || pg_pw == "" || pg_db == "" || pg_port == "" {
		log.Fatal("database credentials are not defined")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		pg_host, pg_user, pg_pw, pg_db, pg_port)

	database.Connect(dsn)

	autoMigrateErr := database.DB.AutoMigrate(
		&model.AxTable{},
		&model.TableField{},
		&model.TableRelation{},
		&model.RelationField{},
		&model.Edt{},
	)

	if autoMigrateErr != nil {
		log.Fatalf("error: %s\n", autoMigrateErr.Error())
	}

	r := server.NewRouter(database.DB)

	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal(err)
	}
}
