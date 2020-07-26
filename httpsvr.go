//Package httpsvr supports a router that understand http method and url params
//(features that standard http_ServeMux lacked).
//This package can be configured to log all pairs of request/response by adding an
//auto-generated requestId to the request_Context. This package also monitors
//number of requests for each handlers, requests duration percentile.
package httpsvr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mywrap/gofast"
	"github.com/mywrap/log"
	"github.com/mywrap/metric"
)

// Server must be inited by calling func NewServer
type Server struct {
	// config defines parameters for running an HTTP server,
	// usually user should set ReadHeaderTimeout, ReadTimeout, WriteTimeout,
	// ReadTimeout and WriteTimeout should be bigger for a file server
	config *http.Server
	// router example usage: router.AddHandler("GET", "/match/:id", func(w,r))
	router         *httprouter.Router
	isEnableLog    bool          // default NewServer set isEnableLog = true
	isEnableMetric bool          // default NewServer set isEnableMetric = true
	Metric         metric.Metric // default is a in-memory metric
}

// NewServer init a Server with my recommended settings.
// For more customizable server, use NewServerWithConf instead.
func NewServer() *Server {
	router := httprouter.New()
	config := NewDefaultConfig()
	config.Handler = router
	s := &Server{
		config:         config,
		isEnableLog:    true,
		isEnableMetric: true,
		router:         router,
		Metric:         metric.NewMemoryMetric(),
	}
	s.AddHandler("GET", "/__metric", s.handleMetric())
	return s
}

// NewServerWithConf is used for turning off log, turning off metric
// or providing a persistent metric instead of in-memory.
// Usually, using simple func NewServer is enough.
func NewServerWithConf(conf *http.Server, isLog bool,
	hasMetric bool, metric0 metric.Metric) *Server {
	if hasMetric && metric0 == nil {
		metric0 = metric.NewMemoryMetric()
	}
	if conf == nil {
		conf = NewDefaultConfig()
	}
	router := httprouter.New()
	conf.Handler = router
	s := &Server{
		config:         conf,
		isEnableLog:    isLog,
		isEnableMetric: hasMetric,
		router:         router,
		Metric:         metric0,
	}
	s.AddHandler("GET", "/__metric", s.handleMetric())
	return s
}

// AddHandler defines the router. Ex: AddHandler("GET", "/", ExampleHandler()).
// The router matches the URL of each incoming request against a list of
// registered path/method patterns and calls the handler for the pattern.
func (s *Server) AddHandler(method string, path string, handler http.HandlerFunc) {
	defer func() { // in case of adding a same handler twice
		if r := recover(); r != nil {
			log.Infof("error when AddHandler: %v", r)
		}
	}()
	// be careful with augmenting handler, example of stack overflow:
	// 	f := func() { log.Println("f called") }
	//	f = func() { f() }
	//	f()

	var augmented1 http.HandlerFunc
	if !s.isEnableMetric {
		augmented1 = handler
	} else {
		metricKey := fmt.Sprintf("%v_%v", path, method)
		augmented1 = func(w http.ResponseWriter, r *http.Request) {
			s.Metric.Count(metricKey)
			beginTime := time.Now()
			handler(w, r)
			s.Metric.Duration(metricKey, time.Since(beginTime))
		}
	}

	var augmented2 http.HandlerFunc
	if !s.isEnableLog {
		augmented2 = augmented1
	} else {
		augmented2 = func(w http.ResponseWriter, r *http.Request) {
			requestId := gofast.GenUUID()
			ctx := context.WithValue(r.Context(), CtxRequestId, requestId)
			query := r.URL.Query().Encode()
			if query != "" {
				query = "?" + query
			}
			log.Condf(s.isEnableLog, "http request %v from %v: %v %v%v",
				requestId, r.RemoteAddr, r.Method, r.URL.Path, query)
			augmented1(w, r.WithContext(ctx))
			log.Condf(s.isEnableLog, "http responded %v to %v: %v %v%v",
				requestId, r.RemoteAddr, r.Method, r.URL.Path, query)
		}
	}

	s.router.HandlerFunc(method, path, augmented2)
}

// ListenAndServe listens on input TCP network address addr
func (s *Server) ListenAndServe(addr string) error {
	s.config.Addr = addr
	return s.config.ListenAndServe()
}

// ListenAndServe listens on the port that defined in s_config_Addr
func (s *Server) ListenAndServe2() error {
	return s.ListenAndServe(s.config.Addr)
}

// NewDefaultConfig is my suggestion of a http server config,
// feel free to modified base on your circumstance
func NewDefaultConfig() *http.Server {
	return &http.Server{
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      20 * time.Minute,
	}
}

// GetUrlParams returns URL parameters from a http request as a map,
// ex: path `/match/:id` has param `id`
func GetUrlParams(r *http.Request) map[string]string {
	params := httprouter.ParamsFromContext(r.Context())
	result := make(map[string]string, len(params))
	if len(params) == 0 {
		return result
	}
	for _, param := range params {
		result[param.Key] = param.Value
	}
	return result
}

//
// utility
//

// Write is a utility to respond body with logging
func (s Server) Write(w http.ResponseWriter, r *http.Request, body string) (
	int, error) {
	w.Header().Set("Content-Type", "text/plain")
	n, err := w.Write([]byte(body))
	if err != nil { // will never happen
		log.Condf(s.isEnableLog, "error WriteJson %v: %v",
			GetRequestId(r), err)
		return n, err
	}
	log.Condf(s.isEnableLog, "http write body %v: %v",
		GetRequestId(r), body)
	return n, nil
}

// WriteJson is a utility to respond body with logging
func (s Server) WriteJson(w http.ResponseWriter, r *http.Request, obj interface{}) (
	int, error) {
	bodyB, err := json.Marshal(obj)
	if err != nil {
		log.Condf(s.isEnableLog, "error WriteJson %v: %v",
			GetRequestId(r), err)
		http.Error(w, err.Error(), 500)
		return 0, err
	}
	w.Header().Set("Content-Type", "application/json")
	n, err := w.Write(bodyB)
	if err != nil { // will never happen
		log.Condf(s.isEnableLog, "error WriteJson %v: %v",
			GetRequestId(r), err)
		return n, err
	}
	log.Condf(s.isEnableLog, "http write body %v: %s",
		GetRequestId(r), bodyB)
	return n, nil
}

// ReadJson is a utility to parse request json body
func (s Server) ReadJson(r *http.Request, outPtr interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	log.Condf(s.isEnableLog, "http request body %v: %s", GetRequestId(r), body)
	err = json.Unmarshal(body, outPtr)
	return err
}

func (s Server) handleMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentMetric := s.Metric.GetCurrentMetric()
		sort.Sort(metric.SortByAveDur(currentMetric))
		beauty, _ := json.MarshalIndent(currentMetric, "", "\t")
		w.Write(beauty)
	}
}

var emptyServer = &Server{isEnableLog: true}

// WriteJson is a utility to respond body with logging
func WriteJson(w http.ResponseWriter, r *http.Request, obj interface{}) (
	int, error) {
	return emptyServer.WriteJson(w, r, obj)
}

// ReadJson is a utility to parse request json body with logging
func ReadJson(r *http.Request, outPtr interface{}) error {
	return emptyServer.ReadJson(r, outPtr)
}

// GetRequestId returns the auto generated unique requestId
func GetRequestId(r *http.Request) string {
	return fmt.Sprintf("%v", r.Context().Value(CtxRequestId))
}

// ctxKeyType is used for avoiding context key conflict
type ctxKeyType string

// CtxRequestId is a internal request id
const CtxRequestId ctxKeyType = "CtxRequestId"
