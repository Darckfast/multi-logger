package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	slogFields key = "slog_fields"
)

var (
	maxQueue     = make(chan int, 5)
	wg           sync.WaitGroup
	API_KEY      = ""
	SERVICE_NAME = ""
)

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	fields := make(map[string]any, record.NumAttrs())

	fields["message"] = record.Message
	fields["level"] = record.Level.String()
	fields["timestamp"] = record.Time.UTC()

	record.Attrs(func(attribute slog.Attr) bool {
		fields[attribute.Key] = attribute.Value.Any()
		return true
	})

	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, attr := range attrs {
			fields[attr.Key] = attr.Value.Any()
		}
	}

	startedAt := fields["startedAt"].(time.Time)
	duration := time.Since(startedAt).Milliseconds()
	fields["duration"] = duration

	jsonBytes, _ := json.Marshal(fields)

	h.l.Println(string(jsonBytes))

	if API_KEY != "" {
		SendLogsHTTP(&record, fields)
	}

	return nil
}

func SendLogsHTTP(record *slog.Record, fields map[string]any) {
	maxQueue <- 1
	wg.Add(1)

	payload := BaselimePayload{
		Message:   record.Message,
		Level:     record.Level.String(),
		Duration:  fields["duration"],
		RequestId: fields["requestId"],
		Timestamp: fields["timestamp"],
		Namespace: fields["namespace"],
	}

	if record.Level == slog.LevelError {
		payload.Error = fields["error"]
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://events.baselime.io/v1/logs", bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-api-key", API_KEY)
	req.Header.Add("x-service", SERVICE_NAME)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer wg.Done()

		client.Do(req)

		<-maxQueue
	}()
}

func NewHandler(
	out io.Writer,
) *Handler {
	h := &Handler{
		Handler: slog.NewJSONHandler(out, nil),
		l: log.New(out, "", 0),
	}

	return h
}

func AppendCtx(parent context.Context, attr slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(slogFields).([]slog.Attr); ok {
		v = append(v, attr)
		return context.WithValue(parent, slogFields, v)
	}

	v := []slog.Attr{}
	v = append(v, attr)
	return context.WithValue(parent, slogFields, v)
}

func SetupContext(opts *SetupOps) (context.Context, *sync.WaitGroup) {
	uid, _ := uuid.NewV7()
	ctx := AppendCtx(context.Background(), slog.String("requestId", uid.String()))
	ctx = AppendCtx(ctx, slog.String("namespace", opts.Namespace))
	ctx = AppendCtx(ctx, slog.Time("startedAt", time.Now()))
	SERVICE_NAME = opts.ServiceName
	API_KEY = opts.ApiKey

	return ctx, &wg
}
