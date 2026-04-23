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
}

func NewWorkerPool(
	workerCount int,
	queueSize int,
	repo *repository.AxTableRepository,
) *WorkerPool {

	wp := &WorkerPool{
		queue: make(chan AxTableRecord, queueSize),
		repo:  repo,
	}

	for i := range workerCount {
		wp.wg.Add(1)
		go wp.work(i)
	}

	return wp
}

func (wp *WorkerPool) work(id int) {
	defer wp.wg.Done()

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

		err := wp.repo.CreateBatch([]model.AxTable{dataAxTable})
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
	}
}

func (wp *WorkerPool) Submit(r AxTableRecord) {
	wp.queue <- r
}

func (wp *WorkerPool) Close() {
	close(wp.queue)
	wp.wg.Wait()
}
