package svc2handler

import (
	"context"
	"net/http"
	"reflect"
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
	if false {
		B.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r.ServeHTTP(newMockWriter(), req)
			}
		})
	} else {
		w := newMockWriter()
		for i := 0; i < B.N; i++ {
			r.ServeHTTP(w, req)
		}
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

type (
	t struct {
		A int
	}

	aSetter interface {
		SetA(a int)
	}
)

func (t *t) SetA(a int) {
	t.A = a
}

func BenchmarkRun(b *testing.B) {
	e := wrapper(func() {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunContext(b *testing.B) {
	e := wrapper(func(context.Context) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunMap(b *testing.B) {
	e := wrapper(func(in interface{}) {
		return
	})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunStruct(b *testing.B) {
	e := wrapper(func(t) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunMultiParam(b *testing.B) {
	e := wrapper(func(t, context.Context) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunStructWithCtx(b *testing.B) {
	e := wrapper(func(context.Context, t) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunMultiParamContext(b *testing.B) {
	e := wrapper(func(context.Context, t, int) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunMultiParam2(b *testing.B) {
	e := wrapper(func(t, t, t, string, int) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunMultiStruct5WithCtx(b *testing.B) {
	e := wrapper(func(context.Context, t, t, t, string, int) {})
	runRequest(b, e, "GET", "")
}

func BenchmarkRunDef(b *testing.B) {
	e := func(writer http.ResponseWriter, request *http.Request) {
		_ = reflect.ValueOf(request.Context())
	}
	runRequest(b, e, "GET", "")
}

func BenchmarkRunCall(b *testing.B) {
	e := reflect.ValueOf(func(ctx context.Context) {})
	v := []reflect.Value{reflect.ValueOf(context.Background())}
	for i := 0; i < b.N; i++ {
		e.Call(v)
	}
}

func BenchmarkRunCallRaw(b *testing.B) {
	e := func(ctx context.Context) {}
	v := context.Background()
	for i := 0; i < b.N; i++ {
		e(v)
	}
}
