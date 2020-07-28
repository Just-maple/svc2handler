package svc2handler

import (
	"net/http"
	"testing"
)

type mockWriter struct {
	headers http.Header
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		http.Header{},
	}
}

func (m *mockWriter) Header() (h http.Header) {
	return m.headers
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockWriter) WriteHeader(int) {}

func runRequest(B *testing.B, r http.HandlerFunc, method, path string) {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	B.ReportAllocs()
	B.ResetTimer()
	//B.RunParallel(func(pb *testing.PB) {
	//	for pb.Next() {
	//		r.ServeHTTP(newMockWriter(), req)
	//	}
	//})
	w := newMockWriter()
	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}
}

type testIO struct {
}

func (t testIO) Response(w http.ResponseWriter, ret interface{}, err error) {
	return
}

func (t testIO) ParamHandler(w http.ResponseWriter, r *http.Request, params []interface{}) (ok bool) {
	return true
}

var wrapper = CreateSvcHandler(&testIO{})

func BenchmarkRun(b *testing.B) {
	e := wrapper(func() {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunDef(b *testing.B) {
	e := func(writer http.ResponseWriter, request *http.Request) {}
	runRequest(b, e, "GET", "")
}
