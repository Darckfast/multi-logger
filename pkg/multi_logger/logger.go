package multilogger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
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
	AXIOM_API_KEY = ""
	SERVICE_NAME  = ""
)

var sendLogsArs SendLogsArgs

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

	if fields["duration"] == nil {
		startedAt := fields[STARTED_AT_KEY].(time.Time)
		duration := time.Since(startedAt).Milliseconds()
		fields["duration"] = duration
	}

	jsonBytes, _ := json.Marshal(fields)
	h.l.Println(string(jsonBytes))
	body, _ := json.Marshal([]any{fields})

	if AXIOM_API_KEY != "" {
		sendLogsArs.body = &body
		SendLogs(sendLogsArs)
	}

	return nil
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

	if r != nil {
		ctx = AppendCtx(ctx, slog.String("query", r.URL.RawQuery))
		ctx = AppendCtx(ctx, slog.String("user-agent", r.UserAgent()))
		ctx = AppendCtx(ctx, slog.String("ip", r.RemoteAddr))
		ctx = AppendCtx(ctx, slog.String("host", r.Host))
		ctx = AppendCtx(ctx, slog.String("method", r.Method))

		requestIp := r.Header.Get("X-Forwarded-For")
		connectingIp := r.Header.Get("CF-Connecting-IP")
		if connectingIp == "" {
			requestIp += connectingIp
		}
		ctx = AppendCtx(ctx, slog.String("x-forwarded-for", requestIp))
		ctx = AppendCtx(ctx, slog.String("country", r.Header.Get("CF-IPCountry")))
		ctx = AppendCtx(ctx, slog.Int64("content-length", r.ContentLength))
		ctx = AppendCtx(ctx, slog.String("content-type", r.Header.Get("content-type")))
		ctx = AppendCtx(ctx, slog.String(NAMESPACE_KEY, r.URL.Path))
	}

	if opts.RequestGen != nil {
		SendLogs = opts.RequestGen
	}

	ctx = AppendCtx(ctx, slog.String("service", opts.ServiceName))
	ctx = AppendCtx(ctx, slog.Time(STARTED_AT_KEY, time.Now()))

	AXIOM_API_KEY = opts.AxiomApiKey

	sendLogsArs.wg = &sync.WaitGroup{}
	sendLogsArs.ctx = ctx
	sendLogsArs.maxQueue = make(chan int, 5)
	sendLogsArs.method = "POST"
	sendLogsArs.url = "https://api.axiom.co/v1/datasets/main/ingest"
	sendLogsArs.bearer = "Bearer " + AXIOM_API_KEY

	return ctx, sendLogsArs.wg
}
