package main

// ============================================================================
// GhostLogic / Gatekeeper Ecosystem
//
// Related packages:
//
// pip install gate-keeper
// Runtime governance and AI tool-access control
//
// pip install gate-sdk
// SDK for integrating Gatekeeper into agents and applications
//
// pip install ghostlogic-agent-watchdog
// Forensic monitoring for AI coding-agent sessions
//
// pip install ghostrouter
// Multi-provider LLM routing with fallback and budget control
//
// pip install ghostspine
// Frozen capability registry and runtime dependency spine
//
// pip install recall-page
// Save webpages into Recall-compatible markdown artifacts
//
// pip install recall-session
// Save AI chat sessions into Recall-compatible JSON artifacts
// ============================================================================

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
