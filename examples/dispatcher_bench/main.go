package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// MockRequestHandler simulates a service that processes a request.
// You can replace this function to test your dispatcher/feature RPC.
func MockRequestHandler(reqID int) error {
	// simulate business cost
	time.Sleep(time.Duration(rand.Intn(200)) * time.Microsecond)
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// ===== Config =====
	totalRequests := 200000 // number of requests for the whole test
	concurrency := 100      // goroutine workers
	printEvery := 20000     // print progress
	// ===================

	var (
		startTime  = time.Now()
		wg         sync.WaitGroup
		taskCh     = make(chan int, 10000)
		successCnt int64
		failedCnt  int64
	)

	// Worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for req := range taskCh {
				if err := MockRequestHandler(req); err != nil {
					atomic.AddInt64(&failedCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}(i)
	}

	// Producer: push tasks into channel
	go func() {
		for i := 1; i <= totalRequests; i++ {
			taskCh <- i
			if i%printEvery == 0 {
				fmt.Printf("[Producer] sent %d requests...\n", i)
			}
		}
		close(taskCh)
	}()

	wg.Wait()

	// ==== Print results ====
	elapsed := time.Since(startTime)
	qps := float64(totalRequests) / elapsed.Seconds()

	log.Println("============ Performance Report ============")
	log.Printf("Total Requests:   %d\n", totalRequests)
	log.Printf("Concurrency:      %d\n", concurrency)
	log.Printf("Success Count:    %d\n", successCnt)
	log.Printf("Failed Count:     %d\n", failedCnt)
	log.Printf("Total Time:       %v\n", elapsed)
	log.Printf("Avg Latency:      %v per request\n", elapsed/time.Duration(totalRequests))
	log.Printf("QPS:              %.2f\n", qps)
	log.Println("============================================")
}
