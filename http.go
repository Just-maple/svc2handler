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
		newParamPool   sync.Pool
	}

	newParam struct {
		i []interface{}
		v []reflect.Value
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
	}
	if v.Kind() != reflect.Func {
		panic("invalid service func")
	}
	switch funcNumOut {
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
		panic("service num out must be one or two")
	}
	for i := 0; i < funcNumIn; i++ {
		ad.types[i] = svcTyp.In(i)
		ad.kinds[i] = ad.types[i].Kind()
	}
	ad.firstIsContext = ad.types[0] == rTypeContext
	return ad.httpHandler(io)
}

func (ad *adapter) httpHandler(io IOController) http.HandlerFunc {
	ad.newParamPool = sync.Pool{
		New: func() interface{} {
			return &newParam{
				i: make([]interface{}, ad.funcNumIn, ad.funcNumIn),
				v: make([]reflect.Value, ad.funcNumIn, ad.funcNumIn),
			}
		},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		newP := ad.newParamPool.Get().(*newParam)
		newParamV := newP.v
		newParam := newP.i
		for i := 0; i < ad.funcNumIn; i++ {
			if i == 0 && ad.firstIsContext {
				continue
			}
			typ := ad.types[i]
			param := reflect.New(typ)
			if ad.kinds[i] == reflect.Ptr {
				param.Elem().Set(reflect.New(typ.Elem()))
			}
			newParamV[i] = param.Elem()
			newParam[i] = param.Interface()
		}
		if ad.firstIsContext {
			newParam = newParam[1:]
		}

		if !io.ParamHandler(w, r, newParam) {
			ad.newParamPool.Put(newP)
			return
		}
		if ad.firstIsContext {
			newParamV[0] = reflect.ValueOf(r.Context())
		}
		ad.retFunc(w, ad.svcV.Call(newParamV))
		ad.newParamPool.Put(newP)
	}
}
