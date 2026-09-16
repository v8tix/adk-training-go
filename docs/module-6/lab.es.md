# Laboratorio 6: Ejecución Programática: Apps y Runners (Go) 🚀

## Objetivo

Vamos más allá de la CLI (módulo 5) y disparamos el agente Support Analyzer desde tu propio código Go — construye un `*runner.Runner` y maneja dos usuarios independientes a través de él, probando el aislamiento de sesiones, exactamente la forma que necesita un backend real.

## Tareas del Laboratorio

### 1. Lee `internal/agents/supportanalyzer/agent.go` 📖

Acá es donde vive ahora la definición del agente — sacada de `cmd/support-analyzer` en este módulo para que más de un programa la pueda construir. `BuildRootAgent(llmModel)` es toda la superficie pública: le das un `model.LLM`, te devuelve un `agent.Agent` listo. Resuelve su propio prompt internamente, así que quien lo llame nunca necesita conocer la clave del caché de prompts.

### 2. Lee `cmd/support-analyzer-runner/main.go` 📖

Sin ningún import de `cmd/launcher` — es un programa Go plano que maneja el agente directamente:

```go
r, _ := runner.NewInMemory("support_analyzer_runner_app", rootAgent)

fmt.Println("--- User A (Alice) ---")
aliceResult, _ := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
fmt.Printf("Agent Response: %s\n", aliceResult)

fmt.Println("\n--- User B (Bob) ---")
bobResult, _ := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
fmt.Printf("Agent Response: %s\n", bobResult)
```

`runOnce` es el helper estándar de este repo para "manejar un mensaje a través del iterador de `Run`, devolver el resultado estructurado final" (mira el README para el patrón completo).

### 3. Ejecútalo ▶️

```bash
go run ./cmd/support-analyzer-runner
```

Salida real y confirmada de este comando exacto:

```
🎫 support-analyzer-runner using qwen3.8:27b
--- User A (Alice) ---
Agent Response: {"category": "billing", "sentiment": "negative", "summary": "The customer reports being overcharged $50 on a charge."}

--- User B (Bob) ---
Agent Response: {"category": "technical", "sentiment": "negative", "summary": "The customer reports that their wifi connection is slow."}
```

Dos análisis distintos y correctos — el reclamo de facturación de Alice categorizado como `billing`, el problema de wifi de Bob como `technical` — ambos servidos por la *misma* instancia de `*runner.Runner`. La redacción va a variar entre ejecuciones (la salida del LLM no es reproducible token por token), pero la forma de categoría/sentimiento y el aislamiento entre los dos usuarios se van a mantener. 🎯

### 4. Verifica que no hay regresión en `cmd/support-analyzer` ✔️

El punto de entrada CLI/launcher de los módulos 4-5 no tiene cambios de comportamiento — ahora solo importa `internal/agents/supportanalyzer` en vez de definir el agente localmente:

```bash
go run ./cmd/support-analyzer console
```

Debería comportarse exactamente igual que antes de este módulo.

## Preguntas de Autorreflexión 🤔
- ¿Por qué se considera que el `runner.Config` de Go fusiona los conceptos `App` y `Runner` de Python, en vez de que le falte una funcionalidad `App`?
- ¿Qué pasaría si `cmd/support-analyzer-runner` reutilizara el mismo `sessionID` para Alice y Bob en vez de usar sesiones separadas?
- En un servidor web Go real (construido sobre `net/http`, por ejemplo), ¿dónde construirías el `*runner.Runner` — dentro de cada handler de request, o una sola vez al arrancar como variable compartida? ¿Por qué esa respuesta coincide con la propia guía de Python para su singleton `Runner`?
- Este módulo sacó `internal/agents/supportanalyzer` específicamente porque Go no puede tener dos `func main()` en un directorio. ¿Qué código de un módulo anterior habría necesitado el mismo tratamiento si hubiera ganado un segundo punto de entrada?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/supportanalyzer/agent.go` (`BuildRootAgent`) y `cmd/support-analyzer-runner/main.go` (`runOnce`, y las dos llamadas a `runOnce` para Alice y Bob) — ese es todo el mecanismo, de principio a fin.
