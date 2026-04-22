package pipeline

import (
	"log"
	"sync"
)

type Record map[string]any

type WorkerPool struct {
	queue chan Record
	wg    sync.WaitGroup
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		queue: make(chan Record, queueSize),
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
		_ = record
	}
}

func (wp *WorkerPool) Submit(r Record) {
	wp.queue <- r
}

func (wp *WorkerPool) Close() {
	close(wp.queue)
	wp.wg.Wait()
	log.Println("all workers finished")
}
