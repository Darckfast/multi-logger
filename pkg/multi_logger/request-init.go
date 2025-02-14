package multilogger

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type SendLogsFunc func(maxQueue chan int, wg *sync.WaitGroup, method, url, bearer string, body *[]byte)

var SendLogs = func(maxQueue chan int, wg *sync.WaitGroup, method, url, bearer string, body *[]byte) {
	maxQueue <- 1
	wg.Add(1)

	req, _ := http.NewRequest(method, url, bytes.NewBuffer(*body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", bearer)

	client := &http.Client{
		Timeout: time.Second * 1,
	}

	go func() {
		defer wg.Done()

		client.Do(req)

		<-maxQueue
	}()
}
