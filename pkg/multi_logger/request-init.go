package multilogger

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"
)

type SendLogsArgs struct {
	Ctx      context.Context
	MaxQueue chan int
	Wg       *sync.WaitGroup
	Method   string
	Url      string
	Bearer   string
	Body     *[]byte
}

type SendLogsFunc func(args SendLogsArgs)

var SendLogs = func(args SendLogsArgs) {
	args.MaxQueue <- 1
	args.Wg.Add(1)

	req, _ := http.NewRequest(args.Method, args.Url, bytes.NewBuffer(*args.Body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", args.Bearer)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer args.Wg.Done()

		client.Do(req)

		<-args.MaxQueue
	}()
}
