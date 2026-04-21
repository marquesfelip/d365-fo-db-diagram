package handlers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProcessPackages(c *gin.Context) {
	var reader io.ReadCloser
	var err error

	body := c.Request.Body

	if c.GetHeader("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		defer reader.Close()
	} else {
		reader = body
	}

	defer body.Close()

	buffer := make([]byte, 32*1024) // 32Kb

	var total int64

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			total += int64(n)
		}

		fmt.Printf("read %d\n", n)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("read error:", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Decompressed bytes received: %d\n", total)

	c.JSON(http.StatusOK, gin.H{
		"bytes_received": total,
	})
}
