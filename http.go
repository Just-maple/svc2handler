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
		paramsPool     *sync.Pool
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
		return ad.emptyParamHandler
	}

	for i := 0; i < ad.funcNumIn; i++ {
		if i == 0 {
			ad.firstIsContext = ad.types[i] == rTypeContext
		}
		ad.types[i] = svcTyp.In(i)
		ad.kinds[i] = ad.types[i].Kind()
	}

	if ad.funcNumIn == 1 && ad.firstIsContext {
		return ad.contextOnlyHandler
	}

	ad.paramsPool = &sync.Pool{
		New: func() interface{} {
			return ad.initParams()
		},
	}

	if !ad.firstIsContext {
		return ad.notContextParamsHandler
	}

	return ad.contextParamsHandler
}

func (ad *adapter) initParams() *paramsCarrier {
	ret := &paramsCarrier{
		values: make([]reflect.Value, ad.funcNumIn),
		params: make([]interface{}, 0, ad.funcNumIn),
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

func (ad *adapter) contextOnlyHandler(w http.ResponseWriter, r *http.Request) {
	ad.retFunc(w, ad.svcV.Call([]reflect.Value{reflect.ValueOf(r.Context())}))
}

func (ad *adapter) emptyParamHandler(w http.ResponseWriter, r *http.Request) {
	ad.retFunc(w, ad.svcV.Call(nil))
}

func (ad *adapter) doHandle(w http.ResponseWriter, r *http.Request, params *paramsCarrier) {
	if len(params.params) != 0 && !ad.io.ParamHandler(w, r, params.params) {
		return
	}
	ad.retFunc(w, ad.svcV.Call(params.values))
}

func (ad *adapter) notContextParamsHandler(w http.ResponseWriter, r *http.Request) {
	var params *paramsCarrier
	if enablePool {
		params = ad.paramsPool.Get().(*paramsCarrier)
		defer ad.paramsPool.Put(params)
	} else {
		params = ad.initParams()
	}

	ad.doHandle(w, r, params)
}

func (ad *adapter) contextParamsHandler(w http.ResponseWriter, r *http.Request) {
	var params *paramsCarrier
	if enablePool {
		params = ad.paramsPool.Get().(*paramsCarrier)
		defer ad.paramsPool.Put(params)
	} else {
		params = ad.initParams()
	}
	params.values[0] = reflect.ValueOf(r.Context())

	ad.doHandle(w, r, params)
}

var enablePool = false

func SetPoolEnable(enable bool) {
	enablePool = enable
}
