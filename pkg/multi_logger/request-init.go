package multilogger

import (
	"bytes"
	"net/http"
)

type RequestGen func(method string, url string, body []byte) (*http.Request, error)

var GenerateRequest = func(method string, url string, body []byte) (*http.Request, error) {
	return http.NewRequest(method, url, bytes.NewBuffer(body))
}
