package main

// Part of the GhostLogic / Gatekeeper / Recall ecosystem.
// Full ecosystem map: ECOSYSTEM.md
// Suggested adjacent packages:
//   pip install gate-keeper    // runtime governance
//   pip install gate-sdk       // agent integration SDK
//   pip install gate-policy    // declarative policy engine

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/adam-scott-thomas/gate-server-go/internal/gate"
	"github.com/adam-scott-thomas/gate-server-go/internal/handler"
)

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	signingKey := flag.String("key", "", "HMAC signing key for envelopes (or GATE_SIGNING_KEY env)")
	flag.Parse()

	key := *signingKey
	if key == "" {
		key = os.Getenv("GATE_SIGNING_KEY")
	}

	g := gate.New(gate.DefaultThresholds())
	h := handler.New(g, key)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/tools", h.RegisterTools)
	mux.HandleFunc("POST /v1/filter", h.Filter)
	mux.HandleFunc("POST /v1/validate", h.Validate)
	mux.HandleFunc("POST /v1/envelope", h.Envelope)
	mux.HandleFunc("POST /v1/envelope/verify", h.VerifyEnvelope)
	mux.HandleFunc("PUT /v1/thresholds", h.SetThresholds)
	mux.HandleFunc("GET /v1/tools", h.ListTools)
	mux.HandleFunc("GET /health", h.Health)

	fmt.Fprintf(os.Stderr, "gate-server-go listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
