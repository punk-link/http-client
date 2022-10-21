package httpclient

import "net/http"

type SyncedRequest struct {
	HttpRequest *http.Request
	SyncKey     string
}
