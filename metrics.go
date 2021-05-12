package main

// metrics module provides various metrics about our server
//
// Copyright (c) 2020 - Valentin Kuznetsov <vkuznet AT gmail dot com>

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

// TotalGetRequests counts total number of GET requests received by the server
var TotalGetRequests uint64

// TotalPostRequests counts total number of POST requests received by the server
var TotalPostRequests uint64

// MetricsLastUpdateTime keeps track of last update time of the metrics
var MetricsLastUpdateTime time.Time

// RPS represents requests per second for a given server
var RPS float64

// RPSPhysical represents requests per second for a given server times number of physical CPU cores
var RPSPhysical float64

// RPSLogical represents requests per second for a given server times number of logical CPU cores
var RPSLogical float64

// NumPhysicalCores represents number of cores in our node
var NumPhysicalCores int

// NumLogicalCores represents number of cores in our node
var NumLogicalCores int

// Memory structure keeps track of server memory
type Memory struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// Mem structure keeps track of virtual/swap memory of the server
type Mem struct {
	Virtual Memory `json:"virtual"` // virtual memory metrics from gopsutils
	Swap    Memory `json:"swap"`    // swap memory metrics from gopsutils
}

// Metrics provide various metrics about our server
type Metrics struct {
	CPU               []float64               `json:"cpu"`               // cpu metrics from gopsutils
	Connections       []net.ConnectionStat    `json:"connections"`       // connections metrics from gopsutils
	Load              load.AvgStat            `json:"load"`              // load metrics from gopsutils
	Memory            Mem                     `json:"memory"`            // memory metrics from gopsutils
	OpenFiles         []process.OpenFilesStat `json:"openFiles"`         // open files metrics from gopsutils
	GoRoutines        uint64                  `json:"goroutines"`        // total number of go routines at run-time
	Uptime            float64                 `json:"uptime"`            // uptime of the server
	GetX509Requests   uint64                  `json:"x509GetRequests"`   // total number of get x509 requests
	PostX509Requests  uint64                  `json:"x509PostRequests"`  // total number of post X509 requests
	GetOAuthRequests  uint64                  `json:"oAuthGetRequests"`  // total number of get requests form OAuth server
	PostOAuthRequests uint64                  `json:"oAuthPostRequests"` // total number of post requests from OAuth server
	GetRequests       uint64                  `json:"getRequests"`       // total number of get requests across all services
	PostRequests      uint64                  `json:"postRequests"`      // total number of post requests across all services
	RPS               float64                 `json:"rps"`               // throughput req/sec
	RPSPhysical       float64                 `json:"rpsPhysical"`       // throughput req/sec using physical cpu
	RPSLogical        float64                 `json:"rpsLogical"`        // throughput req/sec using logical cpu
}

func metrics() Metrics {

	// get cpu and mem profiles
	m, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()
	l, _ := load.Avg()
	c, _ := cpu.Percent(time.Millisecond, true)
	process, perr := process.NewProcess(int32(os.Getpid()))

	// get unfinished queries
	metrics := Metrics{}
	metrics.GoRoutines = uint64(runtime.NumGoroutine())
	virt := Memory{Total: m.Total, Free: m.Free, Used: m.Used, UsedPercent: m.UsedPercent}
	swap := Memory{Total: s.Total, Free: s.Free, Used: s.Used, UsedPercent: s.UsedPercent}
	metrics.Memory = Mem{Virtual: virt, Swap: swap}
	metrics.Load = *l
	metrics.CPU = c
	if perr == nil { // if we got process info
		conn, err := process.Connections()
		if err == nil {
			metrics.Connections = conn
		}
		openFiles, err := process.OpenFiles()
		if err == nil {
			metrics.OpenFiles = openFiles
		}
	}
	metrics.Uptime = time.Since(StartTime).Seconds()
	metrics.GetRequests = TotalGetRequests
	metrics.PostRequests = TotalPostRequests
	if (metrics.GetRequests + metrics.PostRequests) > 0 {
		metrics.RPS = RPS / float64(metrics.GetRequests+metrics.PostRequests)
	}
	if metrics.GetRequests+metrics.PostRequests > 0 {
		metrics.RPSPhysical = RPSPhysical / float64(metrics.GetRequests+metrics.PostRequests)
	}
	if metrics.GetRequests+metrics.PostRequests > 0 {
		metrics.RPSLogical = RPSLogical / float64(metrics.GetRequests+metrics.PostRequests)
	}

	// update time stamp
	MetricsLastUpdateTime = time.Now()

	return metrics
}

// helper function to generate metrics in prometheus format
func promMetrics() string {
	var out string
	data := metrics()
	prefix := "proxy_server"

	// cpu info
	out += fmt.Sprintf("# HELP %s_cpu percentage of cpu used per CPU\n", prefix)
	out += fmt.Sprintf("# TYPE %s_cpu gauge\n", prefix)
	for i, v := range data.CPU {
		out += fmt.Sprintf("%s_cpu{core=\"%d\"} %v\n", prefix, i, v)
	}

	// connections
	var totCon, estCon, lisCon uint64
	for _, c := range data.Connections {
		v := c.Status
		switch v {
		case "ESTABLISHED":
			estCon++
		case "LISTEN":
			lisCon++
		}
	}
	totCon = uint64(len(data.Connections))
	out += fmt.Sprintf("# HELP %s_total_connections\n", prefix)
	out += fmt.Sprintf("# TYPE %s_total_connections gauge\n", prefix)
	out += fmt.Sprintf("%s_total_connections %v\n", prefix, totCon)
	out += fmt.Sprintf("# HELP %s_established_connections\n", prefix)
	out += fmt.Sprintf("# TYPE %s_established_connections gauge\n", prefix)
	out += fmt.Sprintf("%s_established_connections %v\n", prefix, estCon)
	out += fmt.Sprintf("# HELP %s_listen_connections\n", prefix)
	out += fmt.Sprintf("# TYPE %s_listen_connections gauge\n", prefix)
	out += fmt.Sprintf("%s_listen_connections %v\n", prefix, lisCon)

	// load
	out += fmt.Sprintf("# HELP %s_load1\n", prefix)
	out += fmt.Sprintf("# TYPE %s_load1 gauge\n", prefix)
	out += fmt.Sprintf("%s_load1 %v\n", prefix, data.Load.Load1)
	out += fmt.Sprintf("# HELP %s_load5\n", prefix)
	out += fmt.Sprintf("# TYPE %s_load5 gauge\n", prefix)
	out += fmt.Sprintf("%s_load5 %v\n", prefix, data.Load.Load5)
	out += fmt.Sprintf("# HELP %s_load15\n", prefix)
	out += fmt.Sprintf("# TYPE %s_load15 gauge\n", prefix)
	out += fmt.Sprintf("%s_load15 %v\n", prefix, data.Load.Load15)

	// memory virtual
	out += fmt.Sprintf("# HELP %s_mem_virt_total reports total virtual memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_virt_total gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_virt_total %v\n", prefix, data.Memory.Virtual.Total)
	out += fmt.Sprintf("# HELP %s_mem_virt_free reports free virtual memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_virt_free gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_virt_free %v\n", prefix, data.Memory.Virtual.Free)
	out += fmt.Sprintf("# HELP %s_mem_virt_used reports used virtual memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_virt_used gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_virt_used %v\n", prefix, data.Memory.Virtual.Used)
	out += fmt.Sprintf("# HELP %s_mem_virt_pct reports percentage of virtual memory\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_virt_pct gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_virt_pct %v\n", prefix, data.Memory.Virtual.UsedPercent)

	// memory swap
	out += fmt.Sprintf("# HELP %s_mem_swap_total reports total swap memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_swap_total gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_swap_total %v\n", prefix, data.Memory.Swap.Total)
	out += fmt.Sprintf("# HELP %s_mem_swap_free reports free swap memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_swap_free gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_swap_free %v\n", prefix, data.Memory.Swap.Free)
	out += fmt.Sprintf("# HELP %s_mem_swap_used reports used swap memory in bytes\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_swap_used gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_swap_used %v\n", prefix, data.Memory.Swap.Used)
	out += fmt.Sprintf("# HELP %s_mem_swap_pct reports percentage swap memory\n", prefix)
	out += fmt.Sprintf("# TYPE %s_mem_swap_pct gauge\n", prefix)
	out += fmt.Sprintf("%s_mem_swap_pct %v\n", prefix, data.Memory.Swap.UsedPercent)

	// open files
	out += fmt.Sprintf("# HELP %s_open_files reports total number of open file descriptors\n", prefix)
	out += fmt.Sprintf("# TYPE %s_open_files gauge\n", prefix)
	out += fmt.Sprintf("%s_open_files %v\n", prefix, len(data.OpenFiles))

	// go routines
	out += fmt.Sprintf("# HELP %s_goroutines reports total number of go routines\n", prefix)
	out += fmt.Sprintf("# TYPE %s_goroutines counter\n", prefix)
	out += fmt.Sprintf("%s_goroutines %v\n", prefix, data.GoRoutines)

	// uptime
	out += fmt.Sprintf("# HELP %s_uptime reports server uptime in seconds\n", prefix)
	out += fmt.Sprintf("# TYPE %s_uptime counter\n", prefix)
	out += fmt.Sprintf("%s_uptime %v\n", prefix, data.Uptime)

	// x509 requests
	out += fmt.Sprintf("# HELP %s_get_x509_requests reports total number of X509 HTTP GET requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_get_x509_requests counter\n", prefix)
	out += fmt.Sprintf("%s_get_x509_requests %v\n", prefix, data.GetX509Requests)
	out += fmt.Sprintf("# HELP %s_post_x509_requests reports total number of X509 HTTP POST requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_post_x509_requests counter\n", prefix)
	out += fmt.Sprintf("%s_post_x509_requests %v\n", prefix, data.PostX509Requests)

	// oauth requests
	out += fmt.Sprintf("# HELP %s_get_oauth_requests reports total number of OAuth HTTP GET requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_get_oauth_requests counter\n", prefix)
	out += fmt.Sprintf("%s_get_oauth_requests %v\n", prefix, data.GetOAuthRequests)
	out += fmt.Sprintf("# HELP %s_post_oauth_requests reports total number of OAuth HTTP POST requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_post_oauth_requests counter\n", prefix)
	out += fmt.Sprintf("%s_post_oauth_requests %v\n", prefix, data.PostOAuthRequests)

	// total requests
	out += fmt.Sprintf("# HELP %s_get_requests reports total number of HTTP GET requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_get_requests counter\n", prefix)
	out += fmt.Sprintf("%s_get_requests %v\n", prefix, data.GetRequests)
	out += fmt.Sprintf("# HELP %s_post_requests reports total number of HTTP POST requests\n", prefix)
	out += fmt.Sprintf("# TYPE %s_post_requests counter\n", prefix)
	out += fmt.Sprintf("%s_post_requests %v\n", prefix, data.PostRequests)

	// throughput, rps, rps physical cpu, rps logical cpu
	out += fmt.Sprintf("# HELP %s_rps reports request per second average\n", prefix)
	out += fmt.Sprintf("# TYPE %s_rps gauge\n", prefix)
	out += fmt.Sprintf("%s_rps %v\n", prefix, data.RPS)

	out += fmt.Sprintf("# HELP %s_rps_physical_cpu reports request per second average weighted by physical CPU cores\n", prefix)
	out += fmt.Sprintf("# TYPE %s_rps_physical_cpu gauge\n", prefix)
	out += fmt.Sprintf("%s_rps_physical_cpu %v\n", prefix, data.RPSPhysical)

	out += fmt.Sprintf("# HELP %s_rps_logical_cpu reports request per second average weighted by logical CPU cures\n", prefix)
	out += fmt.Sprintf("# TYPE %s_rps_logical_cpu gauge\n", prefix)
	out += fmt.Sprintf("%s_rps_logical_cpu %v\n", prefix, data.RPSLogical)

	return out
}

// helper function that calculates request per second metrics
func getRPS(time0 time.Time) {
	RPS += 1. / time.Since(time0).Seconds()
	RPSLogical += float64(NumLogicalCores) / time.Since(time0).Seconds()
	RPSPhysical += float64(NumPhysicalCores) / time.Since(time0).Seconds()
}
