package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mywrap/httpsvr"
	"github.com/mywrap/log"
)

type Server struct {
	// your server with inited database connection
	*httpsvr.Server
	DatabaseMock     string
	allowCORSOrigins map[string]bool // map key is scheme://host:port
}

func (s *Server) Route() {
	s.Router.GlobalOPTIONS = s.allowCORS(nil)
	s.AddHandler("GET", "/", s.index)
	s.AddHandler("POST", "/login", s.allowCORS(s.login))
	s.AddHandler("GET", "/admin", s.auth(s.hello))
	s.AddHandler("GET", "/exception", s.exception)
	s.AddHandlerNotFound(func(http.ResponseWriter, *http.Request) {})
}

func (s *Server) allowCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, found := s.allowCORSOrigins[origin]; !found {
			w.WriteHeader(http.StatusBadRequest) // unexpected origin
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "*")
		// header Authorization must be listed explicitly, value "*" only
		//counts as a special wildcard value for requests without credentials:
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Headers", "Authorization")

		if r.Header.Get("Access-Control-Request-Method") != "" || handler == nil {
			// browser preflight request
			w.WriteHeader(http.StatusNoContent)
			return
		}
		handler(w, r)
	}
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	s.Write(w, r, `<!DOCTYPE html><html><body>
		<button onclick="onclickLogin()">POST /login</button>
		<script>
			function onclickLogin() {
				fetch("http://localhost:8000/login", {
					method: "POST",
					headers: {"Content-Type": "application/json"}, 
					body: '{"Username": "hoho", "Password": "haha"}',
				}).then(res => {
					console.log("request completed, response:", res);
				});
			}
		</script></body></html>`)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	type LoginR struct{ Username, Password string }
	type LoginW struct{ UserId int }
	var req LoginR
	s.ReadJson(r, &req) // parse request body to a struct
	log.Printf("request data: %+v", req)
	_ = s.DatabaseMock // some query to check username, password
	s.WriteJson(w, r, LoginW{UserId: 123})
}

type reqCtxKey string

const authUser reqCtxKey = "user"

func (s *Server) auth(handle http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerAuth := r.Header.Get("Authorization")
		words := strings.Split(bearerAuth, " ")
		if len(words) != 2 || words[0] != "Bearer" {
			err := errors.New("need header Authorization: Bearer {token}")
			log.Infof("error when http respond %v: %v",
				httpsvr.GetRequestId(r), err)
			http.Error(w, err.Error(), 401)
			return
		}
		userName := words[1]
		ctx := context.WithValue(r.Context(), authUser, userName)
		handle(w, r.WithContext(ctx))
	}
}

func (s *Server) hello(w http.ResponseWriter, r *http.Request) {
	s.WriteJson(w, r, map[string]string{
		"Data": fmt.Sprintf("Hello %v", r.Context().Value(authUser)),
	})
}

func (s *Server) exception(w http.ResponseWriter, r *http.Request) {
	var b *float64
	a := 1 / *b
	s.WriteJson(w, r, map[string]float64{"a": a})
}

func main() {
	s := Server{
		Server:       httpsvr.NewServer(),
		DatabaseMock: "some database connection",
		allowCORSOrigins: map[string]bool{
			"http://127.0.0.1:8000": true, // for testing OPTIONS request
		},
	}
	s.Route()
	port := ":8000"
	url0 := "http://127.0.0.1" + port
	log.Println(url0 + "/")
	log.Println(url0 + "/login POST can be executed by pressing the button on the home page")
	log.Println(url0 + "/admin expect response status 401")
	log.Println(url0 + "/exception expect no response (exception in the server)")
	log.Println(url0 + "/__metric")
	err := s.ListenAndServe(port)
	if err != nil {
		log.Fatalf("error ListenAndServe: %v", err)
	}
}
