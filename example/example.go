package main

import (
	"context"
	"net/http"

	"github.com/Just-maple/svc2handler"
)

type Param struct {
	A int `json:"a"`
	B int `json:"b"`
}

var (
	wrapper = svc2handler.CreateSvcHandler(TestIO)
)

func main() {
	// easy to create a server
	http.HandleFunc("/add", wrapper.Handle(func(ctx context.Context, param Param) (total int) {
		return param.A + param.B
	}))
	panic(http.ListenAndServe("0.0.0.0:80", nil))
}
