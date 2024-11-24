package multilogger

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
	slogFields     key    = "slog_fields"
	REQUEST_ID_KEY string = "requestId"
	NAMESPACE_KEY  string = "namespace"
	MESSAGE_KEY    string = "message"
	LEVEL_KEY      string = "level"
	TIMESTAMP_KEY  string = "timestamp"
	STARTED_AT_KEY string = "startedAt"
	DATA_KEY       string = "data"
)

var (
	maxQueue            = make(chan int, 5)
	wg                  sync.WaitGroup
	BASELIME_API_KEY    = ""
	AXIOM_API_KEY       = ""
	BETTERSTACK_API_KEY = ""
	SERVICE_NAME        = ""
)

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	fields := make(map[string]any, record.NumAttrs())

	fields[MESSAGE_KEY] = record.Message
	fields[LEVEL_KEY] = record.Level.String()
	fields[TIMESTAMP_KEY] = record.Time.UTC()

	record.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, attr := range attrs {
			fields[attr.Key] = attr.Value.Any()
		}
	}

	if fields[STARTED_AT_KEY] == nil {
		timeNow := time.Now()
		fields[STARTED_AT_KEY] = timeNow
		AppendCtx(ctx, slog.Time(STARTED_AT_KEY, timeNow))
	}

	startedAt := fields[STARTED_AT_KEY].(time.Time)
	duration := time.Since(startedAt).Milliseconds()
	fields["duration"] = duration

	jsonBytes, _ := json.Marshal(fields)

	h.l.Println(string(jsonBytes))

	body, _ := json.Marshal([]any{fields})

	if BASELIME_API_KEY != "" {
		SendLogsBaselime(body)
	}

	if AXIOM_API_KEY != "" {
		SendLogsAxiom(body)
	}

	if BETTERSTACK_API_KEY != "" {
		SendLogsBetterStack(body)
	}

	return nil
}

func SendLogsBaselime(body []byte) {
	maxQueue <- 1
	wg.Add(1)

	req, _ := http.NewRequest("POST", "https://events.baselime.io/v1/logs", bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-api-key", BASELIME_API_KEY)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer wg.Done()

		client.Do(req)

		<-maxQueue
	}()
}

func SendLogsBetterStack(body []byte) {
	maxQueue <- 1
	wg.Add(1)

	req, _ := http.NewRequest("POST", "https://in.logs.betterstack.com", bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+BETTERSTACK_API_KEY)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer wg.Done()

		client.Do(req)

		<-maxQueue
	}()
}

func SendLogsAxiom(body []byte) {
	maxQueue <- 1
	wg.Add(1)

	req, _ := http.NewRequest("POST", "https://api.axiom.co/v1/datasets/main/ingest", bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+AXIOM_API_KEY)

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
		l:       log.New(out, "", 0),
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
	ctx := AppendCtx(context.Background(), slog.String(REQUEST_ID_KEY, uid.String()))
	r := opts.Request

	ctx = AppendCtx(ctx, slog.String("query", r.URL.RawQuery))
	ctx = AppendCtx(ctx, slog.String("user-agent", r.UserAgent()))
	ctx = AppendCtx(ctx, slog.String("ip", r.RemoteAddr))
	ctx = AppendCtx(ctx, slog.String("host", r.Host))
	ctx = AppendCtx(ctx, slog.String("method", r.Method))
	ctx = AppendCtx(ctx, slog.String("x-forwarded-for", r.Header.Get("X-Forwarded-For")))
	ctx = AppendCtx(ctx, slog.Int64("content-length", r.ContentLength))
	ctx = AppendCtx(ctx, slog.String("content-type", r.Header.Get("content-type")))
	ctx = AppendCtx(ctx, slog.String("service", opts.ServiceName))
	ctx = AppendCtx(ctx, slog.String(NAMESPACE_KEY, r.URL.Path))
	ctx = AppendCtx(ctx, slog.Time(STARTED_AT_KEY, time.Now()))

	BASELIME_API_KEY = opts.BaselimeApiKey
	AXIOM_API_KEY = opts.AxiomApiKey
	BETTERSTACK_API_KEY = opts.BetterStackApiKey

	return ctx, &wg
}
