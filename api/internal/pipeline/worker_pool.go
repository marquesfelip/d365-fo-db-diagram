package pipeline

import (
	"log"
	"sync"

	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/repository"
)

type WorkerPool struct {
	queue chan RawRecord
	repo  *repository.RawAxTableRepository
	wg    sync.WaitGroup

	batchSize int
}

func NewWorkerPool(
	workerCount int,
	queueSize int,
	repo *repository.RawAxTableRepository,
) *WorkerPool {

	wp := &WorkerPool{
		queue:     make(chan RawRecord, queueSize),
		repo:      repo,
		batchSize: 500,
	}

	for i := range workerCount {
		wp.wg.Add(1)
		go wp.work(i)
	}

	return wp
}

func (wp *WorkerPool) work(id int) {
	defer wp.wg.Done()

	buffer := make([]model.RawAxTable, 0, wp.batchSize)

	flush := func() {
		if len(buffer) == 0 {
			return
		}

		err := wp.repo.CreateBatch(buffer)
		if err != nil {
			log.Fatal(err.Error())
		}

		buffer = buffer[:0]
	}

	for record := range wp.queue {
		rawData := model.RawAxTable{
			Name:    record.Name,
			Model:   record.Model,
			Layer:   &record.Layer,
			Payload: record.Data,
		}

		buffer = append(buffer, rawData)
		if len(buffer) >= wp.batchSize {
			flush()
		}
	}

	flush()
}

func (wp *WorkerPool) Submit(r RawRecord) {
	wp.queue <- r
}

func (wp *WorkerPool) Close() {
	close(wp.queue)
	wp.wg.Wait()
}
