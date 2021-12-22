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
	s.AddHandler("GET", "/", s.index)
	s.AddHandler("OPTIONS", "/login", s.allowCORS(emptyHandler))
	s.AddHandler("POST", "/login", s.login)
	s.AddHandler("GET", "/admin", s.auth(s.hello))
	s.AddHandler("GET", "/exception", s.exception)
	s.AddHandlerNotFound(func(http.ResponseWriter, *http.Request) {})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	s.Write(w, r, "Index page")
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	type LoginR struct{ Password string }
	type LoginW struct{ UserId int }
	var req LoginR
	s.ReadJson(r, &req) // parse request body to a struct
	_ = s.DatabaseMock  // some query to check username, password
	s.WriteJson(w, r, LoginW{UserId: 1})
}

type reqCtxKey string

const authUser reqCtxKey = "user"

func (s Server) auth(handle http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerAuth := r.Header.Get("Authorization")
		words := strings.Split(bearerAuth, " ")
		if len(words) != 2 || words[0] != "Bearer" {
			err := errors.New("need header Authorization: Bearer {token}")
			log.Infof("error when http respond %v: %v",
				httpsvr.GetRequestId(r), err)
			http.Error(w, err.Error(), 500)
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

func (s *Server) allowCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, found := s.allowCORSOrigins[origin]; !found {
			if origin != "" { // normal user uses a normal browser
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("unexpected origin"))
				return
			} else { // probably a developer is debugging
				handler(w, r)
				return
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization") // header Authorization must be listed explicitly
		handler(w, r)
	}
}

func emptyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	s := Server{
		Server:       httpsvr.NewServer(),
		DatabaseMock: "some database connection",
		allowCORSOrigins: map[string]bool{
			"http://localhost:3000": true, // ReactJS app
			"http://127.0.0.1:8000": true, // this server, just for testing OPTIONS request
		},
	}
	s.Route()
	port := ":8000"
	url0 := "http://127.0.0.1" + port
	log.Println(url0 + "/")
	log.Println(url0 + "/__metric")
	log.Println(url0 + "/login")
	log.Println(url0 + "/admin")
	log.Println(url0 + "/exception")
	// curl -X POST --data '{"un": "xyz", "pw": "xyz"}' http://127.0.0.1:8000/echo
	err := s.ListenAndServe(port)
	if err != nil {
		log.Fatalf("error ListenAndServe: %v", err)
	}
}
