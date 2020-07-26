# HTTP server

Package httpsvr supports a router that understand http method and url
params (features that standard http_ServeMux lacked).

This package can be configured to log all pairs of request/response 
by adding an auto-generated requestId to the request_Context. This
package also monitors number of requests for each handlers, requests
duration percentile.

API for adding a handler is similar to standard http ServeMux HandleFunc.

Wrapped [julienschmidt/httprouter](
https://github.com/julienschmidt/httprouter).

## Usage

````go
type Server struct { // your server with inited database connection
	*httpsvr.Server
	DatabaseMock struct{}
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

func main() {
	s := Server{
		Server:       httpsvr.NewServer(),
		DatabaseMock: "some database connection",
	}
	s.AddHandler("POST", "/login", s.login())
	port := ":8000"
	err := s.ListenAndServe(port)
	if err != nil {
		log.Fatalf("error ListenAndServe: %v", err)
	}
}
````

More detail in [example.go](./example/example.go)
