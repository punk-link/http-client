package httpclient

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type HttpClientService[T any] struct {
	config *HttpClientConfig
}

func New[T any](config *HttpClientConfig) HttpClient[T] {
	return &HttpClientService[T]{
		config: config,
	}
}

func (t *HttpClientService[T]) MakeBatchRequest(requests []*http.Request) []T {
	syncedRequests := make([]SyncedRequest, len(requests))
	for i, request := range requests {
		syncedRequests[i] = SyncedRequest{
			HttpRequest: request,
		}
	}

	syncedResults := t.MakeBatchRequestWithSync(syncedRequests)
	results := make([]T, len(syncedResults))
	for i, result := range syncedResults {
		results[i] = result.Result
	}

	return results
}

func (t *HttpClientService[T]) MakeBatchRequestWithSync(syncedHttpRequests []SyncedRequest) []SyncedResult[T] {
	if t.config.IterationStep <= 1 {
		panic("the batch iteration step value must be at least 2 or bigger")
	}

	iterationStep := t.config.IterationStep
	alignedRequests, unalignedRequests := alignSlice(syncedHttpRequests, iterationStep)

	var wg sync.WaitGroup
	chanResults := make(chan SyncedResult[T])

	for i := 0; i < len(alignedRequests); i = i + iterationStep {
		wg.Add(iterationStep)

		for j := 0; j < iterationStep; j++ {
			go t.processRequestAsync(&wg, chanResults, alignedRequests[i+j])
		}

		time.Sleep(t.config.RequestBatchTimeout)
	}

	for _, request := range unalignedRequests {
		wg.Add(1)
		go t.processRequestAsync(&wg, chanResults, request)
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

func (t *HttpClientService[T]) MakeRequest(request *http.Request) (*T, error) {
	if t.config.RequestAttemptsNumber <= 0 {
		panic("the number of request's attempts must be at lease 1 or higher")
	}

	client := &http.Client{}
	var response *http.Response

	var err error
	attemptsLeft := t.config.RequestAttemptsNumber
	for {
		if attemptsLeft == 0 {
			break
		}

		response, err = client.Do(request)
		if err != nil {
			t.config.Logger.LogError(err, err.Error())
			return nil, err
		}

		if response.StatusCode == http.StatusOK {
			break
		}

		if response.StatusCode == http.StatusTooManyRequests {
			t.config.Logger.LogWarn("Web request ends with a status code %v", response.StatusCode)
			time.Sleep(t.getTimeoutDuration(attemptsLeft))
		}

		attemptsLeft--
	}

	return t.unmarshalResponseContent(response)
}

func (t *HttpClientService[T]) getTimeoutDuration(attemptNumber int) time.Duration {
	standardizedMaxJitterInterval := t.config.TimeoutJitterInterval / time.Millisecond

	jit := time.Duration(rand.Intn(int(standardizedMaxJitterInterval))) * time.Millisecond

	base := t.config.TimeoutIntervales[attemptNumber]
	return base + jit
}

func (t *HttpClientService[T]) processRequestAsync(wg *sync.WaitGroup, results chan<- SyncedResult[T], syncedHttpRequest SyncedRequest) {
	defer wg.Done()

	responseContent, err := t.MakeRequest(syncedHttpRequest.HttpRequest)
	if err != nil {
		return
	}

	results <- SyncedResult[T]{
		Result:  *responseContent,
		SyncKey: syncedHttpRequest.SyncKey,
	}
}

func (t *HttpClientService[T]) unmarshalResponseContent(response *http.Response) (*T, error) {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.config.Logger.LogError(err, err.Error())
		return nil, err
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		t.config.Logger.LogError(err, err.Error())
		return nil, err
	}

	return &result, nil
}
