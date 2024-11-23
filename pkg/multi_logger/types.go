package multilogger

import (
	"log"
	"log/slog"
	"net/http"
)

type Handler struct {
	slog.Handler
	l *log.Logger
}

type key string

type BaselimePayload struct {
	Message   string `json:"message"`
	Error     any    `json:"error"`
	RequestId any    `json:"requestId"`
	Namespace any    `json:"namespace"`
	Duration  any    `json:"duration"`
	Timestamp any    `json:"timestamp"`
	Level     string `json:"string"`
}

type SetupOps struct {
	BaselimeApiKey string
	AxiomApiKey    string
	ServiceName    string
	Request        *http.Request
}
