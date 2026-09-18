// Command streaming-agent serves both the Streaming Agent's real /run_live
// WebSocket endpoint and its own hand-written HTML/JS push-to-talk voice
// client from the same origin — a single net/http server, not this repo's
// usual two-cmd/-program split (module-29's cmd/ui-agent +
// cmd/ui-client-server).
//
// That split is not just a style choice here: confirmed live,
// RunLiveHandler (server/adkrest/controllers/runtime.go) creates its own
// websocket.Upgrader with no CheckOrigin override, so it falls back to
// gorilla/websocket's default same-origin check (server.go's
// checkSameOrigin — true only when the request has no Origin header at all,
// or that Origin's host matches the request's own Host). A real browser
// always sends Origin, so any client served from a different origin than
// this agent gets a 403 on every /run_live upgrade attempt, no matter what
// -webui_address or CORS headers say — those only ever governed ordinary
// HTTP responses, never the WebSocket handshake itself. Serving both from
// one process on one port sidesteps the problem entirely by construction:
// Origin always equals Host. This is also why this program bypasses
// cmd/launcher (which owns its own listener with no hook to add static
// routes into it) in favor of building the same minimal net/http server the
// SDK's own reference client uses (google.golang.org/adk/v2/examples/bidi).
//
// A second, independent gotcha confirmed live along the way, moot once you
// bypass the launcher this way but worth knowing if you reach for it
// first (module-29's own shape): the launcher's default "/api" mount wraps
// every request in its own redirectRewriter (cmd/launcher/web/api/api.go),
// which embeds http.ResponseWriter as an interface field and therefore does
// not implement http.Hijacker even though the real underlying writer does —
// gorilla/websocket's own Upgrade does a plain `w.(http.Hijacker)` type
// assertion (server.go), which fails under that wrapper. /run_sse
// (module-29) was unaffected because SSE only needs http.Flusher, which
// redirectRewriter does implement.
//
// This module's own lab invokes it as:
//
//	go run ./cmd/streaming-agent
//
// then opens http://localhost:9095 in a browser.
//
// Unlike every other cmd/ program in this repo, streaming-agent forces
// MODEL_TYPE to llm.ModelTypeVertexAILive regardless of the environment —
// runner.RunLive rejects any model that doesn't expose a *genai.Client (see
// internal/llminternal/base_flow.go's own interface check), which rules out
// Ollama and every other backend this project's llm.Config otherwise
// selects. This is the first module in the whole series with no
// local-only path at all.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/streamingagent"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/server/adkrest/controllers"
	"google.golang.org/adk/v2/session"
)

const addr = ":9095"

// thisFileDir is this source file's own directory, resolved once via
// runtime.Caller — cwd-independent, the same technique established for
// every other cmd/ program's static file serving in this repo (see
// cmd/ui-client-server/main.go).
var thisFileDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}
	cfg := llm.LoadConfig()
	cfg.ModelType = llm.ModelTypeVertexAILive
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🎙️  streaming-agent using %s\n", modelName)

	rootAgent, err := streamingagent.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	controller := controllers.NewRuntimeAPIControllerWithConfig(controllers.RuntimeAPIControllerConfig{
		SessionService: session.InMemoryService(),
		AgentLoader:    agent.NewSingleLoader(rootAgent),
		// AutoCreateSession means the client needs no separate session-
		// creation POST before opening the WebSocket — confirmed live
		// against runner.Runner.getOrCreateSession, unlike module-29's own
		// /run_sse client, which does need that extra step.
		AutoCreateSession: true,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// The WebSocket handshake is always a GET; a method-aware pattern rejects
	// anything else with a 405 instead of trying (and failing) to upgrade it.
	mux.Handle("GET /run_live", controllers.NewErrorHandler(controller.RunLiveHandler))
	staticDir := filepath.Join(thisFileDir, "static")
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	fmt.Printf("🎙️  streaming-agent serving %s on http://localhost%s\n", staticDir, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("run failed: %v", err)
	}
}
