package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	handlers "github.com/marquesfelip/d365-fo-db-diagram/internal/handler"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("error loading .env file: ", err)
	}

	FRONTEND_URL := os.Getenv("FRONTEND_URL")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{FRONTEND_URL},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Enconding"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		api.POST("/process-packages", handlers.ProcessPackages)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	return r
}
