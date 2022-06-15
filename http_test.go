package svc2handler

import (
	"context"
	"net/http"
	"testing"
)

func runTestRequest(t *testing.T, r http.HandlerFunc, method, path string) {
	req, _ := http.NewRequest(method, path, nil)
	req = req.WithContext(context.WithValue(context.Background(), "xxx", 1))
	r.ServeHTTP(newMockWriter(), req)
}

type testIO struct {
}

func (t testIO) Response(w http.ResponseWriter, ret interface{}, err error) {
	return
}

func (t testIO) ParamHandler(w http.ResponseWriter, r *http.Request, params []interface{}) (ok bool) {
	params[0].(*testStruct).B = r.URL.Path
	return true
}

func TestPool(t *testing.T) {
	SetPoolEnable(true)
	wrapper := CreateSvcHandler(&testIO{})
	e := wrapper(func(ctx context.Context, t *testStruct) {
		t.A = 1
	})
	runTestRequest(t, e, "GET", "1")
	runTestRequest(t, e, "GET", "2")
}

func TestPoolCtx(t *testing.T) {
	SetPoolEnable(true)
	wrapper := CreateSvcHandler(&testIO{})
	e := wrapper(func(ctx context.Context) {
		t.Log(ctx.Value("xxx"))
	})
	runTestRequest(t, e, "GET", "1")
	runTestRequest(t, e, "GET", "2")
}
