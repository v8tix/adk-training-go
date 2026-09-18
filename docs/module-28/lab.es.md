# Laboratorio 28: Construyendo un Servidor MCP de "Carrito de Compras" (Go) 🛠️🔌

## Objetivo

Construye tu propio servidor MCP independiente desde cero — un carrito de compras con estado — y luego conecta un agente de ADK a él como cliente, usando exactamente el mismo mecanismo `mcptoolset`/`mcp.CommandTransport` que el módulo 27 ya probó contra un servidor de terceros.

### Prerrequisitos

Ninguno más allá de lo que el módulo 27 ya necesitaba — este laboratorio corre enteramente sobre el valor por defecto local de Ollama, sin credenciales de nube requeridas. Node.js/`npx` tampoco se necesitan: el servidor de este módulo es un programa de Go, no un paquete de Node de terceros.

## Tareas del Laboratorio

### 1. Lee `cmd/cart-mcp-server/main.go`

Un servidor MCP independiente con **cero dependencia de `google.golang.org/adk/v2`** — coincidiendo con el propio `cart_server.py` de Python siendo un script simple, no un programa de ADK. `cart{mu sync.Mutex, items []string}` mantiene el estado; `addItem`/`view` son sus dos handlers de herramienta, registrados vía `mcp.AddTool(server, &mcp.Tool{...}, handler)`. Nota que `AddItemArgs`/`AddItemResult` son simples structs de Go con etiquetas `jsonschema` — sin ningún schema escrito a mano, a diferencia del laboratorio de Python.

### 2. Lee `cmd/cart-mcp-server/main_test.go`

Tests unitarios puros directamente sobre `cart.addItem`/`cart.view` — sin ningún protocolo MCP involucrado, solo llamadas a funciones de Go — probando que los ítems se acumulan en orden, que un carrito nuevo empieza vacío, y que las llamadas concurrentes a `addItem` son genuinamente seguras contra condiciones de carrera (`-race` es lo que realmente atraparía un mutex faltante acá).

### 3. Lee `internal/agents/mcpcart/agent.go`

`BuildRootAgent(llmModel, repoRoot)` lanza `cmd/cart-mcp-server` (vía `go run`, así que sin paso de construcción separado) como su subproceso de servidor MCP, conectado a través de `mcptoolset.New` + `mcp.CommandTransport` — exactamente el mismo mecanismo que el propio `mcpfilesystem` del módulo 27 ya estableció. `RepoRoot()` resuelve la raíz del repositorio vía el propio directorio de `go env GOMOD`, no una suposición de path relativo — una lección directa de un bug real que la propia revisión del módulo 27 atrapó (un path relativo al directorio de trabajo se rompía silenciosamente bajo `go test`, que corre con un directorio de trabajo distinto al de `go run`).

### 4. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/shopping-agent console
```

Sin `.env`, sin API key. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad) — nota las propias líneas de log `[Server]: ...` del servidor intercaladas con la conversación, reenviadas vía `serverCmd.Stderr = os.Stderr`:

```
🛒 shopping-agent using qwen3.8:27b

User -> Please add milk to my cart.
[Server]: Starting Shopping Cart MCP Server...
[Server]: Waiting for a client to connect...
[Server]: added "milk" to the cart
Agent -> Milk has been added to your cart! Let me know if there's anything else you'd like to add.

User -> Also add eggs.
[Server]: added "eggs" to the cart
Agent -> Eggs have been added to your cart! Your cart now contains:

1. Milk
2. Eggs

Anything else you'd like to add?

User -> What is in my shopping cart?
[Server]: client viewed the cart (2 item(s))
Agent -> Your shopping cart currently contains:

1. Milk
2. Eggs

Would you like to add or remove anything?
```

Ese estado — "milk" y "eggs" apareciendo ambos en la llamada final a `view_cart` — vive enteramente en el subproceso del servidor, genuinamente llevado a través de tres turnos separados de la conversación, no recordado por el propio agente.

### 5. Lee `internal/agents/mcpcart/agent_test.go`

`TestBuildRootAgent_Constructs` prueba que la construcción nunca lanza el subproceso (las sesiones MCP se conectan de forma perezosa — el mismo hallazgo que el módulo 27 ya confirmó). `TestShoppingCart_AddsAndViewsItems_{Ollama,Gemini}` dirige la exacta conversación de tres turnos de arriba a través de un `runner.New` real, verificando que ambos nombres reales de ítems aparecen en la respuesta final — la prueba real y estructural del estado entre procesos, no solo mirar la salida de consola.

## Preguntas de Autorreflexión 🤔
- El estado de `cart` es un simple `[]string` en memoria en un proceso de servidor. ¿Qué se rompería si dos estudiantes corrieran su propio `shopping-agent` al mismo tiempo, cada uno lanzando su propio subproceso de servidor? ¿Qué se rompería distinto si de alguna forma compartieran *una sola* instancia de servidor?
- La inferencia automática de schema de `mcp.AddTool` significa que tu struct de Go *es* el schema. ¿Qué necesitarías cambiar del lado de Go si quisieras renombrar un campo JSON sin cambiar el nombre de tu campo de Go?
- ¿Por qué `RepoRoot()` usa `go env GOMOD` en vez de un path relativo como `filepath.Join(thisFileDir, "..", "..", "..")`?
- Este servidor no tiene forma de *quitar* un ítem del carrito. Si agregaras una herramienta `remove_item_from_cart`, ¿necesitaría algo la inferencia automática de schema de `mcp.AddTool` de tu parte más allá de un nuevo struct de argumentos y un handler?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `cmd/cart-mcp-server/main.go` e `internal/agents/mcpcart/agent.go` para el mecanismo real — un servidor independiente con dos handlers de herramienta tipados, y un agente cliente que lo lanza como subproceso.
