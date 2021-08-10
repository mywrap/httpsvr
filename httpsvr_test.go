package httpsvr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHttp(t *testing.T) {
	for _, s := range []*Server{
		NewServer(),
		NewServerWithConf(nil, false, nil),
	} {
		s.AddHandler("GET", "/",
			func(w http.ResponseWriter, r *http.Request) {
				var request struct{ Field0 string }
				ReadJson(r, &request)
				GetUrlParams(r)
				s.Write(w, r, "PONG")
			})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/",
			strings.NewReader(`{"path":"PING"}`))
		handler := s.Router
		handler.ServeHTTP(w, r)
		resBody, _ := ioutil.ReadAll(w.Result().Body)
		if string(resBody) != "PONG" {
			t.Error(resBody)
		}

		// handle 1
		s.AddHandler("GET", "/error",
			func(w http.ResponseWriter, r *http.Request) {
				WriteJson(w, r, map[string]interface{}{"Data": func() {}})
			})
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/error", nil)
		handler.ServeHTTP(w1, r1)
		if w1.Result().StatusCode != http.StatusInternalServerError {
			t.Errorf("expected InternalServerError but %v", w1.Result().Status)
		}

		// handle 2
		type ParamQueryW struct {
			ParamId string
			ParamF2 string
			QueryQ1 string
			QueryQ2 string
		}
		s.AddHandler("GET", "/match/:id",
			func(w http.ResponseWriter, r *http.Request) {
				res := ParamQueryW{
					ParamId: GetUrlParams(r)["id"],
					QueryQ1: r.FormValue("q1"),
					QueryQ2: r.FormValue("q2"),
				}
				s.WriteJson(w, r, res)
			})
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest("GET", "/match/119?q1=lan&q2=dt", nil)
		handler.ServeHTTP(w2, r2)
		bodyB, _ := ioutil.ReadAll(w2.Result().Body)
		var data ParamQueryW
		err := json.Unmarshal(bodyB, &data)
		if err != nil {
			t.Error(err, string(bodyB))
		}
		if data.ParamId != "119" || data.QueryQ1 != "lan" || data.QueryQ2 != "dt" {
			t.Errorf("data: %#v", data)
		}

		if s.isEnableMetric {
			t.Log(s.Metric.GetCurrentMetric())
		}
	}
}

func TestServer_AddHandlerNotFound(t *testing.T) {
	s0 := NewServer()
	w0 := httptest.NewRecorder()
	r0 := httptest.NewRequest("GET", "/not-found", nil)
	s0.Router.ServeHTTP(w0, r0)
	if r, e := w0.Result().StatusCode, 404; r != e {
		t.Errorf("error AddHandlerNotFound: real %v, expected %v", r, e)
	}

	s1 := NewServer()
	s1.AddHandlerNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/not-found", nil)
	s1.Router.ServeHTTP(w1, r1)
	if r, e := w1.Result().StatusCode, 200; r != e {
		t.Errorf("error AddHandlerNotFound: real %v, expected %v", r, e)
	}
}

func _TestServer_ListenAndServeTLS(t *testing.T) {
	s0 := NewServer()
	t.Logf("about to ListenAndServeTLS")
	err := s0.ListenAndServeTLS(":8000",
		`/home/tungdt/docker/deploy_building_contractors/client_react/xaydungcong_com.crt`,
		//`/home/tungdt/docker/deploy_building_contractors/client_react/xaydungcong_com.key`,
		`/home/tungdt/go/src/github.com/daominah/hello_go/crypto_try/myorg0.key`,
	)
	t.Fatal(err)
}
