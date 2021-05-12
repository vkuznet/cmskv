package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
)

// StartTime represents initial time when we started the server
var StartTime time.Time

// DB represents our DB
var DB *badger.DB

func basePath(s string) string {
	if Config.Base != "" {
		if strings.HasPrefix(s, "/") {
			s = strings.Replace(s, "/", "", 1)
		}
		if strings.HasPrefix(Config.Base, "/") {
			return fmt.Sprintf("%s/%s", Config.Base, s)
		}
		return fmt.Sprintf("/%s/%s", Config.Base, s)
	}
	return s
}

// version of the code
var version string

// Info function returns version string of the server
func Info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("git=%s go=%s date=%s", version, goVersion, tstamp)
}

// helper function which provides all handler routes
func handlers() *mux.Router {
	router := mux.NewRouter()
	router.StrictSlash(true) // to allow /route and /route/ end-points

	// visible routes
	router.HandleFunc(basePath("/info"), InfoHandler).Methods("GET")
	router.HandleFunc(basePath("/store"), StoreHandler).Methods("POST")
	router.HandleFunc(basePath("/fetch/{key:.*}"), FetchHandler).Methods("GET")

	// use various middlewares
	router.Use(limitMiddleware)
	router.Use(loggingMiddleware)
	return router
}

// http server implementation
func server() {
	StartTime = time.Now()

	// initialize limiter
	initLimiter(Config.LimiterPeriod)

	// start badger DB
	var err error
	DB, err = badger.Open(badger.DefaultOptions(Config.BadgerDB))
	if err != nil {
		log.Fatal("unable to open badger DB", err)
	}
	log.Println("badger DB", DB)
	defer DB.Close()

	// the request handler
	base := fmt.Sprintf("%s", Config.Base)
	if base == "" {
		base = "/"
	}

	// set our handlers
	http.Handle(base, handlers())

	// start HTTP or HTTPs server based on provided configuration
	addr := fmt.Sprintf(":%d", Config.Port)
	// Start server without user certificates
	log.Printf("Starting HTTP server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
