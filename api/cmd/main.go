package main

import (
	"log"

	"github.com/marquesfelip/d365-fo-db-diagram/internal/server"
)

func main() {
	r := server.NewRouter()

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
