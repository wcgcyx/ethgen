package request

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

func Request(url string, queries []string, duration time.Duration, concurrency int) error {
	if concurrency <= 0 {
		return fmt.Errorf("Concurrency must be positive number, got %v", concurrency)
	}
	actors := make([]*Actor, 0)
	wg := sync.WaitGroup{}
	start := time.Now()
	for i := 0; i < concurrency; i++ {
		subLen := len(queries) / concurrency
		var subQueries []string
		if i != concurrency-1 {
			subQueries = queries[subLen*i : subLen*(i+1)]
		} else {
			subQueries = queries[subLen*i:]
		}
		// New actor
		actor := newActor(url, subQueries, duration/time.Duration(len(subQueries)))
		actors = append(actors, actor)
		wg.Add(1)
		go func() {
			defer wg.Done()
			actor.start()
		}()
	}
	wg.Wait()
	end := time.Now()
	// Report.
	report(start, end, actors)
	return nil
}

type Actor struct {
	url    string
	client *http.Client

	queries []string
	delay   time.Duration

	result  []time.Duration
	succeed int
}

func newActor(url string, queries []string, delay time.Duration) *Actor {
	return &Actor{
		url:     url,
		client:  &http.Client{},
		queries: queries,
		delay:   delay,
		result:  make([]time.Duration, 0),
		succeed: 0,
	}
}

func (a *Actor) start() {
	for i := 0; i < len(a.queries); i++ {
		// request.
		req, err := http.NewRequest("POST", a.url, bytes.NewReader([]byte(a.queries[i])))
		if err != nil {
			fmt.Printf("Fail to generate request: %v\n", err.Error())
		} else {
			req.Header.Set("Content-Type", "application/json")
			start := time.Now()
			resp, err := a.client.Do(req)
			if err != nil {
				fmt.Printf("Fail to request: %v\n", err.Error())
			} else {
				// Add result
				// TODO: Failed request?
				a.result = append(a.result, time.Now().Sub(start))
				if resp.StatusCode == 200 {
					a.succeed++
				}
			}
		}
		time.Sleep(a.delay)
	}
}

func report(start time.Time, end time.Time, actors []*Actor) {
	min := time.Duration(math.MaxInt64)
	max := time.Duration(0)
	total := time.Duration(0)
	count := 0
	succeed := 0
	for _, actor := range actors {
		for _, res := range actor.result {
			if res > max {
				max = res
			}
			if res < min {
				min = res
			}
			total += res
			count++
		}
		succeed += actor.succeed
	}
	if count == 0 {
		fmt.Printf("Performance result at %v: max %v, min %v, avg NA, time taken %v, succeed/total: %v/%v\n", time.Now(), max, min, end.Sub(start), succeed, count)
	} else {
		fmt.Printf("Performance result at %v: max %v, min %v, avg %v, time taken %v, succeed/total: %v/%v\n", time.Now(), max, min, total/time.Duration(count), end.Sub(start), succeed, count)
	}
}
