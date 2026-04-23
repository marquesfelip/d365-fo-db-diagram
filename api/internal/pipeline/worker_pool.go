package pipeline

import (
	"log"
	"sync"

	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"github.com/marquesfelip/d365-fo-db-diagram/internal/repository"
)

type WorkerPool struct {
	queue chan AxTableRecord
	repo  *repository.AxTableRepository
	wg    sync.WaitGroup

	batchSize int
}

func NewWorkerPool(
	workerCount int,
	queueSize int,
	repo *repository.AxTableRepository,
) *WorkerPool {

	wp := &WorkerPool{
		queue:     make(chan AxTableRecord, queueSize),
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

	buffer := make([]model.AxTable, 0, wp.batchSize)

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
		dataAxTable := model.AxTable{
			Name:               record.Name,
			Model:              record.Model,
			Layer:              &record.Layer,
			Extends:            &record.Extends,
			SaveDataPerCompany: record.SaveDataPerCompany,
			TableGroup:         record.TableGroup,
			TableType:          record.TableType,
			PrimaryIndex:       record.PrimaryIndex,
			ReplacementKey:     record.ReplacementKey,
		}

		buffer = append(buffer, dataAxTable)
		if len(buffer) >= wp.batchSize {
			flush()
		}
	}

	flush()
}

func (wp *WorkerPool) Submit(r AxTableRecord) {
	wp.queue <- r
}

func (wp *WorkerPool) Close() {
	close(wp.queue)
	wp.wg.Wait()
}
