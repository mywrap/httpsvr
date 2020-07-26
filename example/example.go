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

type Server struct { // your server with inited database connection
	*httpsvr.Server
	DatabaseMock string
}

func (s *Server) Route() {
	s.AddHandler("GET", "/", s.index())
	s.AddHandler("POST", "/login", s.login())
	s.AddHandler("GET", "/admin", s.auth(s.hello()))
	s.AddHandler("GET", "/exception", s.exception())
}

func (s *Server) index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Write(w, r, "Index page")
	}
}

func (s *Server) login() http.HandlerFunc {
	type LoginR struct{ Password string }
	type LoginW struct{ UserId int }
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginR
		s.ReadJson(r, &req) // parse request body to a struct
		_ = s.DatabaseMock  // some query to check username, password
		s.WriteJson(w, r, LoginW{UserId: 1})
	}
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

func (s *Server) hello() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.WriteJson(w, r, map[string]string{
			"Data": fmt.Sprintf("Hello %v", r.Context().Value(authUser)),
		})
	}
}

func (s *Server) exception() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b *float64
		a := 1 / *b
		s.WriteJson(w, r, map[string]float64{"a": a})
	}
}

func main() {
	s := Server{
		Server:       httpsvr.NewServer(),
		DatabaseMock: "some database connection",
	}
	s.Route()
	port := ":8000"
	url0 := "http://127.0.0.1" + port
	log.Println(url0 + "/")
	log.Println(url0 + "/__metric")
	log.Println(url0 + "/login")
	log.Println(url0 + "/admin")
	log.Println(url0 + "/exception")
	err := s.ListenAndServe(port)
	if err != nil {
		log.Fatalf("error ListenAndServe: %v", err)
	}
}
