package main

import (
	"net/http"

	"github.com/mywrap/httpsvr"
	"github.com/mywrap/log"
)

func main() {
	guiDir := `example_file`
	listeningPort := ":8001"
	s := httpsvr.NewServerWithConf(nil, false, nil)
	s.Router.ServeFiles("/*filepath", http.Dir(guiDir))
	log.Printf("httpsvr GUI serving dir %v on http://127.0.0.1%v",
		guiDir, listeningPort)
	err := s.ListenAndServe(listeningPort)
	if err != nil {
		log.Fatal(err)
	}
}
