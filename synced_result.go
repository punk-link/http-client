package httpclient

type SyncedResult[T any] struct {
	Result  T
	SyncKey string
}
