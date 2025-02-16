package multilogger

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"
)

type SendLogsArgs struct {
	ctx      context.Context
	maxQueue chan int
	wg       *sync.WaitGroup
	method   string
	url      string
	bearer   string
	body     *[]byte
}

type SendLogsFunc func(args SendLogsArgs)

var SendLogs = func(args SendLogsArgs) {
	args.maxQueue <- 1
	args.wg.Add(1)

	req, _ := http.NewRequest(args.method, args.url, bytes.NewBuffer(*args.body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", args.bearer)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer args.wg.Done()

		client.Do(req)

		<-args.maxQueue
	}()
}
