package svc2handler

import (
	"context"
	"net/http"
	"reflect"
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
	for i := 0; i < funcNumIn; i++ {
		ad.types[i] = svcTyp.In(i)
		ad.kinds[i] = ad.types[i].Kind()
	}
	if len(ad.types) > 0 {
		ad.firstIsContext = ad.types[0] == rTypeContext
	}
	return ad.httpHandler
}

func (ad *adapter) httpHandler(w http.ResponseWriter, r *http.Request) {
	if ad.funcNumIn == 0 {
		ad.retFunc(w, ad.svcV.Call(nil))
		return
	}

	if ad.funcNumIn == 1 && ad.firstIsContext {
		ad.retFunc(w, ad.svcV.Call([]reflect.Value{reflect.ValueOf(r.Context())}))
		return
	}

	newParamV := make([]reflect.Value, ad.funcNumIn, ad.funcNumIn)
	newParam := make([]interface{}, 0, ad.funcNumIn)
	for i := 0; i < ad.funcNumIn; i++ {
		if i == 0 && ad.firstIsContext {
			newParamV[i] = reflect.ValueOf(r.Context())
			continue
		}
		param := reflect.New(ad.types[i])
		if ad.kinds[i] == reflect.Ptr {
			param.Elem().Set(reflect.New(ad.types[i].Elem()))
		}
		newParamV[i] = param.Elem()
		newParam = append(newParam, param.Interface())
	}
	if len(newParam) == 0 || !ad.io.ParamHandler(w, r, newParam) {
		return
	}
	ad.retFunc(w, ad.svcV.Call(newParamV))
}
