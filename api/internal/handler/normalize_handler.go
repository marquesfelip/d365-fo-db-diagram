package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/normalizer"
	"gorm.io/gorm"
)

type NormalizeHandler struct {
	normalizer *normalizer.Normalizer
}

func NewNormalizeHandler(db *gorm.DB) *NormalizeHandler {
	return &NormalizeHandler{
		normalizer: normalizer.New(db),
	}
}

func (h *NormalizeHandler) Handle(ctx *gin.Context) {
	n, err := h.normalizer.Run()
	if err != nil {
		log.Printf("normalization error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"records_normalized": n})
}
