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
	if Config.Base == "" {
		router.HandleFunc("/info", InfoHandler).Methods("GET")
		router.HandleFunc("/store", StoreHandler).Methods("POST")
		router.HandleFunc("/fetch/{key:.*}", FetchHandler).Methods("GET")
		router.HandleFunc("/", IndexHandler).Methods("GET")
	} else {
		base := Config.Base
		if !strings.HasSuffix(base, "/") {
			base = fmt.Sprintf("%s/", Config.Base)
		}
		subrouter := router.PathPrefix(base).Subrouter()
		//         subrouter.StrictSlash(true) // to allow /route and /route/ end-points
		subrouter.HandleFunc("/info", InfoHandler).Methods("GET")
		subrouter.HandleFunc("/store", StoreHandler).Methods("POST")
		subrouter.HandleFunc("/fetch/{key:.*}", FetchHandler).Methods("GET")
		subrouter.HandleFunc("/", IndexHandler).Methods("GET")

	}

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
	base := Config.Base
	if !strings.HasSuffix(base, "/") {
		base = fmt.Sprintf("%s/", Config.Base)
	}

	// set our handlers
	http.Handle(base, handlers())

	// start HTTP or HTTPs server based on provided configuration
	addr := fmt.Sprintf(":%d", Config.Port)
	// Start server without user certificates
	log.Printf("Starting HTTP server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
