# Laboratorio 25: Construyendo un Plugin de Alertas y Probando Trazado Real (Go) 🔭📊

## Objetivo

Construye un Plugin que observe a un agente en busca de errores consecutivos de herramientas, escalando una alerta y reiniciando al recuperarse — luego prueba que el trazado real y consciente del grafo de OpenTelemetry del SDK realmente funciona, en dos niveles de rigor.

## Tareas del Laboratorio

### 1. Lee `internal/agents/observability/tools.go`

`riskyOperation` devuelve un `error` de Go genuino (`ErrSimulatedFailure`) cuando se le pide fallar — no un resultado estructurado `{"status": "error"}` como el de `calculator.divide`. `OnToolErrorCallback` solo se dispara ante un retorno de error real, y este módulo trata exactamente de observar ese camino.

### 2. Lee `internal/agents/observability/alerting_plugin.go`

`alertTracker` es un struct simple con dos métodos — `onToolError` (marca el turno como con error, incrementa un contador, registra una alerta que escala a CRÍTICA a los 3 fallos consecutivos, y devuelve un resultado con gracia para que el agente no se caiga) y `onEvent` (reinicia el contador una vez que un turno termina limpio, vía `event.IsFinalResponse()`). `newAlertingPlugin` envuelve a ambos vía `plugin.New` — sin interfaz que implementar, solo firmas de función que coinciden.

### 3. Lee `internal/agents/observability/agent.go`

Nota que el plugin **no** está conectado aquí — `BuildRootAgent` solo conecta la herramienta. El plugin se registra por separado, a nivel del Runner/launcher, en `cmd/observability-agent/main.go`. Esta separación es todo el punto de un Sistema de Plugins.

### 4. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/observability-agent console
```

Salida real y confirmada de una conversación de 4 turnos (tres fallos, luego una recuperación limpia; razonamiento del modelo de pensamiento recortado para legibilidad):

```
🔭 observability-agent using qwen3.8:27b

User -> Please call risky_operation and make it FAIL.
⚠️  ALERT: tool "risky_operation" failed (1 consecutive so far): simulated failure
Agent -> The operation failed as requested. It returned a simulated failure error.

User -> Please call risky_operation and make it FAIL.
⚠️  ALERT: tool "risky_operation" failed (2 consecutive so far): simulated failure
Agent -> The operation failed as you requested. It returned a simulated failure error.

User -> Please call risky_operation and make it FAIL.
🚨 CRITICAL ALERT: tool "risky_operation" has failed 3 times consecutively (simulated failure)
Agent -> The operation failed as requested — it returned a "simulated failure" error.

User -> Please call risky_operation without making it fail.
Agent -> The operation completed successfully this time. No failure occurred.
```

Tres alertas escalando, luego silencio en el turno limpio — la propia lógica de reinicio del plugin, observada en vivo.

### 5. Lee `internal/agents/observability/alerting_plugin_test.go`

Tests unitarios puros, sin LLM: escalada en el umbral, el valor de retorno con gracia (sin error), reinicio solo después de un turno final limpio, y un evento no-final siendo un no-op. Se pasa `nil` para los parámetros `agent.Context`/`agent.InvocationContext` en estos tests — ambos métodos de callback solo usan sus argumentos `tool`/`event`/`error`, así que no se necesita ningún doble de prueba para los tipos de contexto en absoluto.

### 6. Lee `internal/agents/observability/agent_test.go`

`TestAlertingPlugin_EscalatesOnConsecutiveErrors_{Ollama,Gemini}` prueba el mecanismo *real* del Plugin, no solo la lógica de callback a nivel unitario ya probada arriba: construye un `runner.New` real con el plugin real conectado en `PluginConfig.Plugins`, corre tres llamadas separadas a `Run()`, y luego lee el propio `alertTracker.errorCount` del plugin directamente (vía el valor de retorno del tracker de `newAlertingPlugin`) para confirmar que la intercepción real ocurrió — no interpretada de la salida de consola o del texto del modelo.

### 7. Lee `internal/agents/observability/telemetry_test.go`

Una función de test, dos subtests, compartiendo una sola configuración de telemetría (ver el propio comentario superior del archivo para saber por qué — llamar a `SetGlobalOtelProviders` dos veces en un proceso no funciona como esperarías):

- `in-memory-attributes` — siempre corre, sin necesitar Docker. Captura un span real `invoke_agent` vía un exportador en memoria y revisa su atributo `gen_ai.agent.name`.
- `real-otlp-export-to-jaeger` — inicia un contenedor real `jaegertracing/all-in-one` vía Testcontainers, exporta un span real hacia él sobre OTLP gRPC real, y luego consulta la propia API HTTP de Jaeger para confirmar que llegó. Se salta limpiamente si Docker no está disponible.

Corre ambos:

```bash
go test ./internal/agents/observability/... -run TestTelemetry_GraphAwareTracing -v
```

Salida real y confirmada de este comando exacto (con Docker disponible):

```
=== RUN   TestTelemetry_GraphAwareTracing
=== RUN   TestTelemetry_GraphAwareTracing/in-memory-attributes
=== RUN   TestTelemetry_GraphAwareTracing/real-otlp-export-to-jaeger
--- PASS: TestTelemetry_GraphAwareTracing (14.53s)
    --- PASS: TestTelemetry_GraphAwareTracing/in-memory-attributes (0.00s)
    --- PASS: TestTelemetry_GraphAwareTracing/real-otlp-export-to-jaeger (0.00s)
PASS
```

**Un bug real que este test detectó mientras se escribía:** el propio método `Shutdown` del exportador en memoria borra sus spans registrados como efecto secundario — llamar a `providers.Shutdown()` (necesario para vaciar el *otro* exportador, el OTLP) antes de leer los spans del exportador en memoria los borró en silencio. La solución fue leer los spans en memoria primero, y luego apagar. Ver los propios comentarios del archivo de test para la historia completa.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `alertTracker.onToolError` devuelve un resultado no nulo en vez del error que recibió? ¿Qué le pasaría a la ejecución del agente si devolviera el error sin cambios?
- El test en vivo de `agent_test.go` lee `alertTracker.errorCount` directamente en vez de revisar la salida de consola en busca del texto "🚨 CRITICAL ALERT". ¿Por qué es esa una prueba más sólida?
- Ambos subtests de telemetría comparten una sola llamada a `telemetry.New`/`SetGlobalOtelProviders()` en vez de que cada uno tenga la suya. ¿Qué saldría mal si no lo hicieran?
- `-otel_to_cloud=true` está documentado pero nunca se usa realmente en los propios tests de este módulo. ¿Por qué no, y qué necesitarías para probarlo de verdad?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/observability/alerting_plugin.go` y `agent.go` para el mecanismo real — los dos métodos de un struct simple, envueltos por `plugin.New`, registrados por separado del propio agente en `cmd/observability-agent/main.go`.
