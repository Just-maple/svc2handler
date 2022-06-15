package svc2handler

import (
	"context"
	"net/http"
	"reflect"
	"sync"
)

var (
	rTypeContext = reflect.TypeOf(new(context.Context)).Elem()
	rTypeError   = reflect.TypeOf(new(error)).Elem()
)

type (
	SvcHandler func(svc interface{}) http.HandlerFunc

	IOController interface {
		Response(w http.ResponseWriter, ret interface{}, err error)

		ParamHandler(w http.ResponseWriter, r *http.Request, params []interface{}) (ok bool)
	}

	adapter struct {
		svcV           reflect.Value
		funcNumIn      int
		funcNumOut     int
		firstIsContext bool
		types          []reflect.Type
		retFunc        func(w http.ResponseWriter, values []reflect.Value)
		kinds          []reflect.Kind
		io             IOController
	}
)

func CreateSvcHandler(svr IOController) SvcHandler {
	return func(svc interface{}) http.HandlerFunc {
		return HandleSvcWithIO(svr, svc)
	}
}

func (w SvcHandler) Handle(svc interface{}) http.HandlerFunc {
	return w(svc)
}

type paramsCarrier struct {
	values []reflect.Value
	params []interface{}
}

func HandleSvcWithIO(io IOController, svc interface{}) http.HandlerFunc {
	v := reflect.ValueOf(svc)
	svcTyp := v.Type()
	funcNumOut := svcTyp.NumOut()
	funcNumIn := svcTyp.NumIn()
	ad := adapter{
		svcV:       v,
		funcNumIn:  funcNumIn,
		funcNumOut: svcTyp.NumOut(),
		types:      make([]reflect.Type, funcNumIn, funcNumIn),
		kinds:      make([]reflect.Kind, funcNumIn, funcNumIn),
		io:         io,
	}
	if v.Kind() != reflect.Func {
		panic("invalid service func")
	}
	switch funcNumOut {
	case 0:
		ad.retFunc = func(w http.ResponseWriter, values []reflect.Value) {
			io.Response(w, nil, nil)
		}
	case 1:
		if svcTyp.Out(0) != rTypeError {
			ad.retFunc = func(w http.ResponseWriter, values []reflect.Value) {
				io.Response(w, values[0].Interface(), nil)
			}
		} else {
			ad.retFunc = func(w http.ResponseWriter, values []reflect.Value) {
				err, _ := values[0].Interface().(error)
				io.Response(w, nil, err)
			}
		}
	case 2:
		if svcTyp.Out(1) != rTypeError {
			panic("service last out must be error")
		}
		ad.retFunc = func(w http.ResponseWriter, values []reflect.Value) {
			err, _ := values[1].Interface().(error)
			io.Response(w, values[0].Interface(), err)
		}
	default:
		panic("service num out must be 0 ~ 2")
	}

	if ad.funcNumIn == 0 {
		return func(w http.ResponseWriter, r *http.Request) {
			ad.retFunc(w, ad.svcV.Call(nil))
		}
	}

	for i := 0; i < ad.funcNumIn; i++ {
		ad.types[i] = svcTyp.In(i)
		ad.kinds[i] = ad.types[i].Kind()
	}

	if len(ad.types) > 0 {
		ad.firstIsContext = ad.types[0] == rTypeContext
	}

	if ad.funcNumIn == 1 && ad.firstIsContext {
		return func(w http.ResponseWriter, r *http.Request) {
			ad.retFunc(w, ad.svcV.Call([]reflect.Value{reflect.ValueOf(r.Context())}))
		}
	}

	paramsNum := ad.funcNumIn
	if ad.firstIsContext {
		paramsNum -= 1
	}

	initParams := func() *paramsCarrier {
		ret := &paramsCarrier{
			values: make([]reflect.Value, ad.funcNumIn),
			params: make([]interface{}, 0, paramsNum),
		}
		for i := 0; i < ad.funcNumIn; i++ {
			if i == 0 && ad.firstIsContext {
				continue
			}
			param := reflect.New(ad.types[i])
			if ad.kinds[i] == reflect.Ptr {
				param.Elem().Set(reflect.New(ad.types[i].Elem()))
			}
			ret.values[i] = param.Elem()
			ret.params = append(ret.params, param.Interface())
		}
		return ret
	}

	paramsPool := sync.Pool{
		New: func() interface{} {
			return initParams()
		},
	}

	if !ad.firstIsContext {
		return func(w http.ResponseWriter, r *http.Request) {
			var params *paramsCarrier
			if enablePool {
				params = paramsPool.Get().(*paramsCarrier)
				defer paramsPool.Put(params)
			} else {
				params = initParams()
			}
			if len(params.params) != 0 && !ad.io.ParamHandler(w, r, params.params) {
				return
			}
			ad.retFunc(w, ad.svcV.Call(params.values))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var params *paramsCarrier
		if enablePool {
			params = paramsPool.Get().(*paramsCarrier)
			defer paramsPool.Put(params)
		} else {
			params = initParams()
		}
		params.values[0] = reflect.ValueOf(r.Context())
		if len(params.params) != 0 && !ad.io.ParamHandler(w, r, params.params) {
			return
		}
		ad.retFunc(w, ad.svcV.Call(params.values))
	}
}

var enablePool = false

func EnablePool() {
	enablePool = true
}
