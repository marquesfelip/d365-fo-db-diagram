package handler

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Record map[string]any

func ProcessPackages(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()

	var reader io.Reader = body

	if c.GetHeader("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		defer gz.Close()
		reader = gz
	}

	scanner := bufio.NewScanner(reader)

	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var count int64

	for scanner.Scan() {
		line := scanner.Bytes()

		var record Record

		if err := json.Unmarshal(line, &record); err != nil {
			log.Println("invalid json:", err)
			continue
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		log.Println("scanner error:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	log.Printf("Processed records: %d\n", count)

	c.JSON(http.StatusOK, gin.H{
		"records_processed": count,
	})

}
