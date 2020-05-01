package main

import (
	"encoding/json"
	"net/http"

	"github.com/Just-maple/svc2handler"
)

var TestIO svc2handler.IOController = testController{}

type (
	testController struct {
	}

	Ret struct {
		Error string      `json:"error,omitempty"`
		Data  interface{} `json:"data,omitempty"`
	}
)

func (t testController) Response(w http.ResponseWriter, ret interface{}, err error) {
	var res Ret
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	ec := json.NewEncoder(w)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res.Data = ret
	if err = ec.Encode(res); err != nil {
		res.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		_ = ec.Encode(res)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (t testController) ParamHandler(w http.ResponseWriter, r *http.Request, params []interface{}) (ok bool) {
	paramLen := len(params)
	switch {
	case paramLen >= 1:
		// bind your param
		switch r.Method {
		case http.MethodGet:
		//	todo:bind param from query
		default:
			_ = json.NewDecoder(r.Body).Decode(params[0])
		}
	}
	return true
}
