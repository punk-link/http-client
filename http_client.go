package httpclient

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func MakeBatchRequestWithSyncKeys[T any](config *HttpClientConfig, syncedHttpRequests []SyncedRequest) []SyncedResult[T] {
	if config.IterationStep <= 1 {
		panic("the batch iteration step value must be at least 2 or bigger")
	}

	iterationStep := config.IterationStep
	alignedRequests, unalignedRequests := alignSlice(syncedHttpRequests, iterationStep)

	var wg sync.WaitGroup
	chanResults := make(chan SyncedResult[T])

	for i := 0; i < len(alignedRequests); i = i + iterationStep {
		wg.Add(iterationStep)

		for j := 0; j < iterationStep; j++ {
			go processRequestAsync(&wg, chanResults, config, alignedRequests[i+j])
		}

		time.Sleep(config.RequestBatchTimeout)
	}

	for _, request := range unalignedRequests {
		wg.Add(1)
		go processRequestAsync(&wg, chanResults, config, request)
	}

	go func() {
		wg.Wait()
		close(chanResults)
	}()

	syncedResults := make([]SyncedResult[T], 0)
	for result := range chanResults {
		syncedResults = append(syncedResults, result)
	}

	return syncedResults
}

func MakeBatchRequest[T any](config *HttpClientConfig, requests []*http.Request) []T {
	syncedRequests := make([]SyncedRequest, len(requests))
	for i, request := range requests {
		syncedRequests[i] = SyncedRequest{
			HttpRequest: request,
		}
	}

	syncedResults := MakeBatchRequestWithSyncKeys[T](config, syncedRequests)
	results := make([]T, len(syncedResults))
	for i, result := range syncedResults {
		results[i] = result.Result
	}

	return results
}

func MakeRequest[T any](config *HttpClientConfig, request *http.Request, result *T) error {
	if config.RequestAttemptsNumber <= 0 {
		panic("the number of request's attempts must be at lease 1 or higher")
	}

	client := &http.Client{}
	var response *http.Response

	var err error
	attemptsLeft := config.RequestAttemptsNumber
	for {
		if attemptsLeft == 0 {
			break
		}

		response, err = client.Do(request)
		if err != nil {
			config.Logger.LogError(err, err.Error())
			return err
		}

		if response.StatusCode == http.StatusOK {
			break
		}

		if response.StatusCode == http.StatusTooManyRequests {
			config.Logger.LogWarn("Web request ends with a status code %v", response.StatusCode)
			time.Sleep(getTimeoutDuration(config, attemptsLeft))
		}

		attemptsLeft--
	}

	return unmarshalResponseContent(config, response, &result)
}

func processRequestAsync[T any](wg *sync.WaitGroup, results chan<- SyncedResult[T], config *HttpClientConfig, syncedHttpRequest SyncedRequest) {
	defer wg.Done()

	var responseContent T
	err := MakeRequest(config, syncedHttpRequest.HttpRequest, &responseContent)
	if err != nil {
		return
	}

	results <- SyncedResult[T]{
		Result:  responseContent,
		SyncKey: syncedHttpRequest.SyncKey,
	}
}

func getTimeoutDuration(config *HttpClientConfig, attemptNumber int) time.Duration {
	standardizedMaxJitterInterval := config.TimeoutJitterInterval / time.Millisecond

	jit := time.Duration(rand.Intn(int(standardizedMaxJitterInterval))) * time.Millisecond

	base := config.TimeoutIntervales[attemptNumber]
	return base + jit
}

func unmarshalResponseContent[T any](config *HttpClientConfig, response *http.Response, result *T) error {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		config.Logger.LogError(err, err.Error())
		return err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		config.Logger.LogError(err, err.Error())
		return err
	}

	return nil
}
