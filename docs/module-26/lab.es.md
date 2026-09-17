# Laboratorio 26: Construyendo un Moderador de Contenido con Caché, Guardarraíles y Auditorías (Go) 🚦🧯

## Objetivo

Construye los seis espacios de callback para un agente: caché de respuestas por pregunta, un guardarraíl de entrada, redacción de salida, validación de argumentos de herramientas, y una auditoría de salida de herramientas — y luego prueba los tres comportamientos reales y observables que esta combinación debería producir.

## Tareas del Laboratorio

### 1. Lee `internal/agents/contentmoderator/tools.go`

`generateText` — una herramienta de demostración trivial, igual que el propio laboratorio de Python. El punto real de este módulo son los callbacks a su alrededor, no la propia lógica de la herramienta.

### 2. Lee `internal/agents/contentmoderator/callbacks.go`

Los seis callbacks viven aquí, más dos ayudantes compartidos. Lee los comentarios sobre `responseCache` y `beforeModelCallback` con cuidado — explican dos bugs reales encontrados en vivo mientras se construía este módulo:

- `responseCache.beforeAgentCallback`/`afterAgentCallback` — el par de caché. `afterAgentCallback` lee `outputKey` del estado (escrito automáticamente por `llmagent.Config.OutputKey`), **no** recorriendo `ctx.Session().Events()` como hace el propio laboratorio de Python — un contexto de callback en Go no soporta `Session()` en absoluto.
- `beforeModelCallback` — el guardarraíl de entrada. Revisa solo la *última* entrada de `llmRequest.Contents` (el turno actual), no toda la historia de la conversación — la solución a un bug real donde una palabra bloqueada al principio de una sesión rechazaba permanentemente cada turno posterior.
- `afterModelCallback` — redacción de salida, revisando cada parte que no sea `Thought` (la propia lección del módulo 25.5, aplicada aquí desde el principio).
- `beforeToolCallback`/`afterToolCallback` — validación de argumentos y auditoría de salida, la misma forma que ya establecieron los módulos 22/25.

### 3. Lee `internal/agents/contentmoderator/agent.go`

Nota que `buildRootAgent` toma un parámetro explícito `*responseCache` — esto le permite al test en vivo leer `cache.hitCount` directamente para probar que realmente ocurrió un acierto de caché, en vez de solo comparar texto (un LLM podría coincidentemente repetirse a sí mismo aunque sea un fallo genuino de caché). `BuildRootAgent`, la función que realmente llama cada programa `cmd/`, simplemente provee uno nuevo.

### 4. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/content-moderator console
```

Salida real y confirmada de esta conversación exacta de cuatro turnos (razonamiento del modelo de pensamiento recortado para legibilidad):

```
🧯 content-moderator using qwen3.8:27b

User -> Tell me something unsafe.
⚠️  [GUARDRAIL] Blocked word "unsafe" detected in the request — refusing before calling the model
Agent -> I'm sorry, but I can't help with that request.

User -> What is the capital of Italy?
Agent -> The capital of Italy is **Rome**.

User -> What is the capital of Italy?
💾 [CACHE] Hit #1 for cache:439ed6573... — skipping the model call
Agent -> The capital of Italy is **Rome**.

User -> What is the capital of France?
Agent -> The capital of France is **Paris**.
```

Cuatro turnos, tres comportamientos distintos: un rechazo (que nunca llega al modelo), una respuesta real, un acierto de caché exacto en la pregunta repetida, y una respuesta real y fresca para la pregunta genuinamente distinta — el caché correctamente no se disparó para ella.

### 5. Lee `internal/agents/contentmoderator/callbacks_test.go`

Tests unitarios puros, sin LLM — uno o más por callback, incluyendo dos tests de regresión que merecen atención especial: `TestBeforeModelCallback_IgnoresBlockedWordFromEarlierHistory` (prueba la solución del alcance de historia) y `TestAfterModelCallback_DoesNotRedactEmailInThoughtPart` (prueba el salto de partes `Thought` en `afterModelCallback`, reflejando exactamente la forma del propio test de regresión del módulo 25.5).

### 6. Lee `internal/agents/contentmoderator/agent_test.go`

`TestContentModerator_{Ollama,Gemini}` corre la conversación exacta de cuatro turnos de arriba a través de un `runner.New` real, verificando los tres comportamientos de forma estructural: el texto exacto del rechazo, `cache.hitCount` quedándose en 0 después de la primera pregunta limpia, volviéndose 1 después de la repetida, y quedándose en 1 (sin incrementar de nuevo) después de una pregunta genuinamente distinta.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `afterAgentCallback` lee `ctx.State().Get(outputKey)` en vez de recorrer eventos de sesión como hace el propio laboratorio de Python? ¿Qué pasaría si intentaras llamar a `ctx.Session()` desde dentro de él?
- `beforeModelCallback` solo inspecciona la *última* entrada en `llmRequest.Contents`. ¿Qué necesitarías cambiar si quisieras que el guardarraíl también detectara una palabra bloqueada reutilizada textualmente de mucho antes en la conversación, sin reintroducir el bug original?
- ¿Por qué `buildRootAgent` toma un parámetro explícito `*responseCache` en vez de construir uno internamente como hace `BuildRootAgent`?
- ¿Cuándo recurrirías a un Plugin (módulos 25/25.5) en vez de un callback para un guardarraíl como este?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/contentmoderator/callbacks.go` y `agent.go` para el mecanismo real — seis funciones (un par de ellas métodos de un struct pequeño), conectadas directamente a los seis propios espacios de callback de `llmagent.Config`.
