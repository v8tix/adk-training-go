package observability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/telemetry"
)

// jaegerTestServiceName is set explicitly via telemetry.WithResource so the
// Jaeger query in the "real-otlp-export-to-jaeger" subtest doesn't have to
// guess the OTel SDK's own default "unknown_service:<binary name>" scheme.
const jaegerTestServiceName = "observability-jaeger-test"

// TestTelemetry_GraphAwareTracing proves Go's own graph-aware tracing —
// internal/telemetry/node_tracing.go starts a real "invoke_agent <name>"
// span (confirmed by reading its source this session), tagged with a
// gen_ai.agent.name attribute matching adk-python's own node_tracing
// semantic conventions — at two levels of rigor, in two subtests:
//
//   - "in-memory-attributes": always runs, no Docker needed. Proves the
//     span/attribute mechanism itself via an in-memory exporter.
//   - "real-otlp-export-to-jaeger": a real jaegertracing/all-in-one
//     container receives spans over a real OTLP gRPC connection, and
//     Jaeger's own HTTP query API confirms the span genuinely arrived — a
//     materially stronger proof than the in-memory exporter alone. Skips
//     (never fails) if Docker/Testcontainers can't start a container,
//     matching this repo's own established pattern
//     (internal/infrastructure/redissession's newTestRedisClient).
//
// Both subtests share ONE telemetry.New/SetGlobalOtelProviders call and one
// *sdktrace.TracerProvider with both span processors attached — deliberately:
// internal/telemetry's own span-emitting code resolves its tracer from the
// global OTel registry into a package-level var, bound the first time a real
// provider is installed in this process. Calling SetGlobalOtelProviders a
// second time (as two separate top-level test functions each doing their
// own setup would) does NOT redirect that already-bound tracer — confirmed
// live this session: splitting this into two independent test functions
// made the second one's spans silently never reach its own exporter.
func TestTelemetry_GraphAwareTracing(t *testing.T) {
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	ctx := t.Context()

	inMemoryExporter := tracetest.NewInMemoryExporter()
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceNameKey.String(jaegerTestServiceName)))
	if err != nil {
		t.Fatalf("resource.New() error = %v", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(inMemoryExporter)))
	providers, err := telemetry.New(ctx, telemetry.WithTracerProvider(tp))
	if err != nil {
		t.Fatalf("telemetry.New() error = %v", err)
	}
	providers.SetGlobalOtelProviders()
	t.Cleanup(func() {
		if err := providers.Shutdown(context.Background()); err != nil {
			t.Logf("telemetry shutdown: %v", err)
		}
	})

	var jaegerQueryEndpoint string
	jaegerAvailable := false
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "jaegertracing/all-in-one:1.60",
			ExposedPorts: []string{"4317/tcp", "16686/tcp"},
			WaitingFor:   wait.ForListeningPort("16686/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Logf("Jaeger container unavailable (%v) — the real-otlp-export-to-jaeger subtest will skip; in-memory-attributes still runs", err)
	} else {
		t.Cleanup(func() {
			if err := container.Terminate(context.Background()); err != nil {
				t.Logf("terminating Jaeger container: %v", err)
			}
		})
		host, hostErr := container.Host(ctx)
		otlpPort, portErr := container.MappedPort(ctx, "4317/tcp")
		queryEndpoint, queryErr := container.PortEndpoint(ctx, "16686/tcp", "http")
		if hostErr == nil && portErr == nil && queryErr == nil {
			otlpEndpoint := fmt.Sprintf("%s:%s", host, otlpPort.Port())
			otlpExporter, expErr := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(otlpEndpoint), otlptracegrpc.WithInsecure())
			if expErr == nil {
				tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(otlpExporter))
				jaegerQueryEndpoint = queryEndpoint
				jaegerAvailable = true
			} else {
				t.Logf("otlptracegrpc.New() error = %v — the real-otlp-export-to-jaeger subtest will skip", expErr)
			}
		} else {
			t.Logf("reading Jaeger container endpoints failed (host=%v port=%v query=%v) — the real-otlp-export-to-jaeger subtest will skip", hostErr, portErr, queryErr)
		}
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	llmModel, _, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}
	obsAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	r, err := runner.NewInMemory("observability_telemetry_test_app", obsAgent)
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}
	if _, err := askObservabilityAgent(ctx, r, "test_user", "test_session", "Please call risky_operation without making it fail."); err != nil {
		t.Fatalf("askObservabilityAgent() error = %v", err)
	}

	// Captured BEFORE Shutdown deliberately: tracetest.InMemoryExporter's
	// own Shutdown calls Reset() internally, clearing every span it holds
	// (confirmed by reading its source this session, after this exact
	// ordering mistake made the in-memory subtest see zero spans while the
	// Jaeger one — unaffected, since spans already crossed the wire —
	// still passed). NewSimpleSpanProcessor exports synchronously at
	// OnEnd, so the spans are already here; Shutdown below exists only to
	// flush the OTLP exporter before querying Jaeger.
	capturedSpans := inMemoryExporter.GetSpans()

	if err := providers.Shutdown(ctx); err != nil {
		t.Fatalf("telemetry Shutdown() (forces the OTLP exporter's pending span out before querying Jaeger) error = %v", err)
	}

	t.Run("in-memory-attributes", func(t *testing.T) {
		var found *tracetest.SpanStub
		for _, span := range capturedSpans {
			if strings.HasPrefix(span.Name, "invoke_agent ") {
				s := span
				found = &s
				break
			}
		}
		if found == nil {
			t.Fatalf("no \"invoke_agent \" span captured; got spans: %+v", capturedSpans)
		}

		var sawAgentName bool
		for _, attr := range found.Attributes {
			if string(attr.Key) == "gen_ai.agent.name" && attr.Value.AsString() == "monitored_agent" {
				sawAgentName = true
			}
		}
		if !sawAgentName {
			t.Errorf("span %q attributes = %+v, want a gen_ai.agent.name = %q attribute", found.Name, found.Attributes, "monitored_agent")
		}
	})

	t.Run("real-otlp-export-to-jaeger", func(t *testing.T) {
		if !jaegerAvailable {
			t.Skip("skipping: Jaeger container/OTLP exporter was not available — see the parent test's log")
		}

		tracesURL := fmt.Sprintf("%s/api/traces?service=%s&limit=20", jaegerQueryEndpoint, jaegerTestServiceName)
		var found bool
		for attempt := 0; attempt < 10 && !found; attempt++ {
			resp, err := http.Get(tracesURL)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if strings.Contains(string(body), "invoke_agent") {
					found = true
					break
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !found {
			t.Errorf("Jaeger's query API at %s never reported a trace containing \"invoke_agent\" — the real OTLP export did not arrive", tracesURL)
		}
	})
}
