package httpclient

import "net/http"

type HttpClient[T any] interface {
	MakeBatchRequest(requests []*http.Request) []T
	MakeBatchRequestWithSync(syncedHttpRequests []SyncedRequest) []SyncedResult[T]
	MakeRequest(request *http.Request) (*T, error)
}
