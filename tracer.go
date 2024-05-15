package main

import (
	"golang.org/x/exp/trace"
	"log"
	"net/http"
)

var fr *trace.FlightRecorder

func EnableTrace() error {
	fr = trace.NewFlightRecorder()
	if err := fr.Start(); err != nil {
		log.Println("start flight recorder error:", err)
		return err
	}
	return nil
}

func HandleTracer(w http.ResponseWriter, r *http.Request) {
	log.Println("request tracer from:", r)
	w.Write([]byte("ok"))
}
