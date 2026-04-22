package handler

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/pipeline"
)

type Record map[string]any

func IngestHandler(ctx *gin.Context) {
	body := ctx.Request.Body
	defer body.Close()

	var reader io.Reader = body

	if ctx.GetHeader("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(body)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		defer gz.Close()
		reader = gz
	}

	// if ctx.GetHeader("Content-Encoding") != "gzip" {
	// 	ctx.JSON(http.StatusBadRequest, gin.H{"message": "Content-Encoding is not in gzip"})
	// 	return
	// }
	//
	// gz, err := gzip.NewReader(body)
	// if err != nil {
	// 	ctx.JSON(http.StatusBadRequest, gin.H{
	// 		"message": fmt.Sprintf(err.Error()),
	// 	})
	// 	return
	// }
	//
	// defer gz.Close()
	// reader = gz

	workers := runtime.NumCPU() * 2
	queueSize := 1000

	wp := pipeline.NewWorkerPool(workers, queueSize)

	scanner := bufio.NewScanner(reader)

	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var count int64

	for scanner.Scan() {
		line := scanner.Bytes()

		var record pipeline.Record

		if err := json.Unmarshal(line, &record); err != nil {
			log.Println("invalid json:", err)
			continue
		}
		wp.Submit(record)
		count++
	}

	if err := scanner.Err(); err != nil {
		log.Println("scanner error:", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	wp.Close()

	log.Printf("Processed records: %d\n", count)

	ctx.JSON(http.StatusOK, gin.H{
		"records_processed": count,
	})

}
