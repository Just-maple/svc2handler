package svc2handler

import (
	"context"
	"net/http"
	"reflect"
	"sync"
)

var (
	rTypeContextPtr = reflect.TypeOf(new(context.Context))
	rTypeContext    = rTypeContextPtr.Elem()
	rTypeError      = reflect.TypeOf(new(error)).Elem()
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

		resetParams []reflect.Value
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
	ctxPtr *context.Context
}

func HandleSvcWithIO(io IOController, svc interface{}) http.HandlerFunc {
	var (
		rv         = reflect.ValueOf(svc)
		svcTyp     = rv.Type()
		funcNumOut = svcTyp.NumOut()
		funcNumIn  = svcTyp.NumIn()
		ad         = adapter{
			svcV:       rv,
			funcNumIn:  funcNumIn,
			funcNumOut: svcTyp.NumOut(),
			types:      make([]reflect.Type, funcNumIn, funcNumIn),
			kinds:      make([]reflect.Kind, funcNumIn, funcNumIn),
			io:         io,
		}
	)

	if rv.Kind() != reflect.Func {
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
		ad.types[i] = svcTyp.In(i)
		if i == 0 {
			ad.firstIsContext = ad.types[i] == rTypeContext
		}
		ad.kinds[i] = ad.types[i].Kind()
		if ad.kinds[i] == reflect.Ptr {
			ad.types[i] = ad.types[i].Elem()
		}
	}

	ad.paramsPool = &sync.Pool{
		New: func() interface{} {
			return ad.initParams()
		},
	}

	if ad.funcNumIn == 1 && ad.firstIsContext {
		return ad.contextOnlyHandler
	}

	ad.resetParams = ad.initParams().values
	for i := range ad.resetParams {
		if i == 0 && ad.firstIsContext {
			continue
		}
		if ad.kinds[i] == reflect.Ptr {
			ad.resetParams[i] = ad.resetParams[i].Elem()
		}
	}

	if !ad.firstIsContext {
		return ad.notContextParamsHandler
	}

	return ad.contextParamsHandler
}

func (ad *adapter) initParams() *paramsCarrier {
	paramsNum := ad.funcNumIn
	if ad.firstIsContext {
		paramsNum -= 1
	}

	ret := &paramsCarrier{
		values: make([]reflect.Value, ad.funcNumIn),
		params: make([]interface{}, paramsNum),
	}

	for i, j := 0, 0; i < ad.funcNumIn; i++ {
		if i == 0 && ad.firstIsContext {
			ret.ctxPtr = new(context.Context)
			ret.values[i] = reflect.ValueOf(ret.ctxPtr).Elem()
			continue
		}

		ret.values[i] = reflect.New(ad.types[i])
		ret.params[j] = ret.values[i].Interface()
		j++

		if ad.kinds[i] != reflect.Ptr {
			ret.values[i] = ret.values[i].Elem()
		}
	}
	return ret
}

func (ad *adapter) contextOnlyHandler(w http.ResponseWriter, r *http.Request) {
	c := ad.paramsPool.Get().(*paramsCarrier)
	*c.ctxPtr = r.Context()
	ad.retFunc(w, ad.svcV.Call(c.values))
	ad.paramsPool.Put(c)
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
		params = ad.getParamsFromPool()
		defer ad.paramsPool.Put(params)
	} else {
		params = ad.initParams()
	}

	ad.doHandle(w, r, params)
}

func (ad *adapter) getParamsFromPool() (params *paramsCarrier) {
	params = ad.paramsPool.Get().(*paramsCarrier)
	for i, v := range params.values {
		if i == 0 && ad.firstIsContext {
			continue
		}
		if ad.kinds[i] == reflect.Ptr {
			v.Elem().Set(ad.resetParams[i])
		} else {
			v.Set(ad.resetParams[i])
		}
	}
	return
}

func (ad *adapter) contextParamsHandler(w http.ResponseWriter, r *http.Request) {
	var params *paramsCarrier
	if enablePool {
		params = ad.getParamsFromPool()
		defer ad.paramsPool.Put(params)
	} else {
		params = ad.initParams()
	}
	*params.ctxPtr = r.Context()
	ad.doHandle(w, r, params)
	return
}

var enablePool = false

func SetPoolEnable(enable bool) {
	enablePool = enable
}
