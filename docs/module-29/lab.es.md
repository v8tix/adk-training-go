# Laboratorio 29: Construyendo un UI de Chat Personalizado Simple (Go) 💬🌐

## Objetivo

Construye un cliente de chat HTML/JavaScript independiente, respaldado por la propia API REST de este repositorio, demostrando exactamente lo que los componentes prearmados de AG-UI abstraen: la mecánica cruda de solicitud/respuesta y streaming SSE.

### Prerrequisitos

Ninguno más allá de lo que cada módulo anterior ya necesitaba — este laboratorio corre enteramente sobre el valor por defecto local de Ollama, sin credenciales de nube requeridas.

## Tareas del Laboratorio

### 1. Lee `internal/agents/uiagent/agent.go`

Un agente deliberadamente trivial, sin herramientas — la verdadera lección de este módulo es el cliente hablándole, no lo que el propio agente puede hacer.

### 2. Lee `cmd/ui-agent/main.go`

Conexión estándar de launcher, relevante acá solo con `api` (sin `webui`, sin `console` — el propio cliente personalizado de este módulo es el "UI"). Córrelo:

```bash
go run ./cmd/ui-agent web --port=9093 api -webui_address http://localhost:9094
```

`-webui_address` es la propia bandera de CORS de `api` (confirmado en el propio `cmd/launcher/web/api/api.go` del SDK fijado) — establecida al propio origen del cliente estático (puerto 9094, abajo) para que las llamadas `fetch` cross-origin del navegador tengan éxito.

### 3. Lee `cmd/ui-agent/main_test.go`

Construye `adkrest.NewServer` directamente (sin launcher, sin CLI) envuelto en `httptest.NewServer` — un servidor HTTP real, en el mismo proceso. `TestRunSSE_StreamsBothThoughtAndFinalParts` dirige exactamente el mismo flujo de creación-de-sesión + `/run_sse` que hará el cliente del navegador, y prueba que el propio riesgo central de este módulo es real, no hipotético: el stream SSE en vivo genuinamente contiene una parte `"thought":true` **y** una respuesta final real, en la misma respuesta. Esta es la propia prueba de regresión del lado del servidor de este módulo; la propia lógica de filtrado de JavaScript del cliente se verifica en vivo en un navegador real en su lugar (Paso 5) — este repositorio no tiene un test runner de JavaScript para probarla unitariamente de forma directa.

### 4. Lee `cmd/ui-client-server/static/index.html`

El cliente de chat real, escrito a mano. Tres cosas para notar respecto a la propia versión del laboratorio de Python:
- Cada endpoint tiene el prefijo `/api` (`API_BASE = 'http://localhost:9093/api'`) — el propio prefijo de path por defecto de la API REST de Go.
- El cuerpo de la solicitud a `/run_sse` usa camelCase (`appName`, `userId`, `sessionId`, `newMessage`) — confirmado contra las propias etiquetas struct de `RunAgentRequest` de este SDK.
- El bucle de parseo de SSE explícitamente salta cualquier parte con `part.thought` verdadero antes de agregar su texto — sin esto, el chat mostraría al usuario el rastro de razonamiento crudo del modelo.

### 5. Corré la pila completa — dos procesos, un navegador 🖥️

**Terminal 1 (servidor del agente):**
```bash
go run ./cmd/ui-agent web --port=9093 api -webui_address http://localhost:9094
```

**Terminal 2 (servidor del cliente estático):**
```bash
go run ./cmd/ui-client-server
```

Luego abrí `http://localhost:9094` en un navegador y enviá un mensaje.

**Verificación real y confirmada de esta configuración exacta** — dirigida con `chromedp` (un driver real de Chrome headless para Go, `github.com/chromedp/chromedp`), no solo un click manual: navegó a la página, escribió "Say hello in exactly three words.", envió el formulario, y sondeó el DOM hasta que el div del mensaje del asistente tuvo texto real.

```
=== Rendered assistant message ===
Hello to you
===================================
✅ No obvious reasoning-trace markers found in the rendered text — thought-filtering appears to work.
```

La propia respuesta de `/run_sse` del servidor para ese turno exacto genuinamente contenía una parte `thought: true` (ver el test del Paso 3) — la página renderizada correctamente muestra solo la respuesta real de tres palabras, probando que la propia lógica de filtrado del cliente realmente funciona de punta a punta, no solo aisladamente.

### Punto de Control

- [ ] `go test ./cmd/ui-agent/... -race` pasa
- [ ] Un navegador real (o un chequeo con Chrome headless) muestra solo la respuesta final — sin texto de razonamiento filtrado — después de enviar un mensaje

## Preguntas de Autorreflexión 🤔
- Los Server-Sent Events transmiten texto al cliente a medida que se genera. ¿Cómo se sentiría para un usuario un endpoint de chat tradicional (sin streaming), comparado con esto?
- El cliente de este laboratorio genera un `sessionId` nuevo en cada carga de página. ¿Qué necesitarías cambiar para que una conversación sobreviva a un refresco de página?
- `cmd/ui-agent/main_test.go` construye `adkrest.NewServer` directamente en vez de pasar por `cmd/launcher`. ¿Por qué eso significa que el test golpea `/run_sse` en vez de `/api/run_sse`, y qué te dice eso sobre de dónde viene realmente el prefijo `/api`?
- El Protocolo AG-UI reemplazaría la mayoría del propio JavaScript de `static/index.html` con un puñado de componentes prearmados de React. ¿Qué exactamente estarías sacrificando al adoptarlo — y qué ganarías?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/uiagent/agent.go`, `cmd/ui-agent/main.go`, y `cmd/ui-client-server/static/index.html` para el mecanismo real — un agente trivial, un launcher de API REST, y un cliente de chat escrito a mano que le habla vía `/api/run_sse`.
