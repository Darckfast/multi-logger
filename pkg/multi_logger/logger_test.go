package multilogger

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetupContext(t *testing.T) {
	opt := &SetupOps{
		Namespace:   "/route/test",
		ApiKey:      "api_key",
		ServiceName: "test-service",
	}

	ctx, wg := SetupContext(opt)

	assert.NotNil(t, ctx)
	assert.NotNil(t, wg)

	attrs, ok := ctx.Value(slogFields).([]slog.Attr)

	assert.True(t, ok)

	fields := make(map[string]any, len(attrs))

	for _, attr := range attrs {
		fields[attr.Key] = attr.Value.Any()
	}

	requestId := fields["requestId"]
	namespace := fields["namespace"]
	startedAt := fields["startedAt"]

	assert.NotEmpty(t, requestId)
	assert.Equal(t, "/route/test", namespace)
	assert.Equal(t, "api_key", API_KEY)
	assert.Equal(t, "test-service", SERVICE_NAME)
	assert.NotNil(t, startedAt)
	assert.IsType(t, time.Time{}, startedAt)
	assert.IsType(t, &sync.WaitGroup{}, wg)
}

func TestNewHandlerWithNoHTTP(t *testing.T) {
	opt := &SetupOps{
		Namespace:   "/route/test",
		ServiceName: "test-service",
	}

	handler := NewHandler(os.Stdout)
	ctx, _ := SetupContext(opt)

	logger := slog.New(handler)
	logger.InfoContext(ctx, "test", "key1", "value1", "key2", "value2")
}
