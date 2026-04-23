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
	"github.com/marquesfelip/d365-fo-db-diagram/internal/repository"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *gin.Engine {
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

	r.SetTrustedProxies([]string{"localhost"})

	AxTableRepo := repository.NewAxTableRepository(db)
	ingestHandler := handlers.NewIngestHandler(AxTableRepo)

	api := r.Group("/api")
	{
		api.POST("/ingest", ingestHandler.Handle)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	return r
}
