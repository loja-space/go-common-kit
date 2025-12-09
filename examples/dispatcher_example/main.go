package main

import (
	"context"
	"fmt"
	"time"

	"github.com/loja-space/go-common-kit/pkg/dispatcher"
)

func main() {
	d := dispatcher.NewRequestDispatcher(10, func(req *dispatcher.Request) *dispatcher.Response {
		time.Sleep(100 * time.Millisecond)
		return &dispatcher.Response{
			Req:    req,
			Result: fmt.Sprintf("processed: %v", req.Payload),
		}
	})

	defer d.Close()

	req := &dispatcher.Request{RequestID: 1, Payload: "hello"}
	fut, _ := d.Send(req)

	resp, _ := fut.Wait(context.Background())
	fmt.Println(resp.Result)
}
