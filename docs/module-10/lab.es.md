# Laboratorio 10: Construyendo un Agente "Memoria" con Herramientas con Estado (Go) 🧠

## Objetivo

Construye un agente que recuerde el nombre de un usuario a través de turnos, usando `agent.Context.State()` para darle a una de sus herramientas memoria real y persistente.

## Tareas del Laboratorio

### 1. Lee `internal/agents/memory/tools.go`

Dos handlers: `storeName` escribe en `ctx.State()` bajo la clave `"user_name"`; `recallName` la lee de vuelta, devolviendo `"Stranger"` cuando todavía no se ha guardado nada (`errors.Is(err, session.ErrStateKeyNotExist)`). Fíjate en el tipo de parámetro de `recallName`, `RecallNameArgs struct{}` — vacío, no omitido, ya que `functiontool.New` necesita un struct o mapa incluso para una herramienta que no toma datos.

### 2. Lee `internal/agents/memory/agent.go`

Misma forma que `internal/agents/calculator/agent.go` — dos llamadas a `functiontool.New`, adjuntas vía `Tools`. Sin backend forzado; este agente está feliz con el default local, igual que la calculadora.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/memory console
```

Sin `.env`, sin API key. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad):

```
🧠 memory using qwen3.8:27b

User -> Hi, I'm Mario.
Agent -> Hi Mario! Great to meet you. How can I help you today?

User -> What is my name?
Agent -> Your name is Mario! Is there anything else I can help you with today?
```

Dos turnos separados, misma sesión — la segunda respuesta prueba que `recall_name` realmente leyó de vuelta lo que `store_name` escribió en el primer turno, no que el modelo simplemente recordó "Mario" del texto crudo del chat (la sección `# Constraints` de la instrucción exige usar la herramienta de cualquier forma — ver `internal/agents/memory/agent_test.go` para el test estructural real de esto). ✨

### 4. Lee `internal/agents/memory/tools_test.go`

`TestStoreName` y `TestRecallName` — tests unitarios puros, sin LLM. Usan un `fakeState` escrito a mano más `agent.StrictContextMock` (el test double propio del SDK para `agent.Context`) en vez de levantar un agente real — la base rápida y determinística de la pirámide de tests de este módulo.

### 5. Lee `internal/agents/memory/agent_test.go`

`TestMemory_RemembersNameAcrossTurns_Ollama` y `_Gemini` — ambos llaman al agente **dos veces**, vía dos llamadas separadas a `Run()` contra el mismo ID de sesión, y verifican que la respuesta de la segunda llamada mencione "Mario". Esto es deliberadamente *no* una llamada a `Run()` con dos mensajes — el punto es probar que el estado sobrevive *entre* fronteras de `Run()`, que es exactamente la gran idea de este módulo.

## Preguntas de Autorreflexión 🤔
- ¿Por qué es más confiable guardar datos en `agent.Context.State()` que simplemente apoyarse en el historial de chat propio del LLM?
- ¿Qué pasaría si dos sesiones diferentes guardaran un nombre bajo la clave `"user_name"`? (Pista: las sesiones se aíslan automáticamente — revisa el parámetro de ID de sesión de `runner.NewInMemory` en `agent_test.go`.)
- ¿Cómo extenderías este agente para recordar algo más, como un color favorito? ¿Qué necesitarías agregar, y dónde?
- `agent.StrictContextMock` hace panic en cualquier método que no sobrescribas. ¿Por qué es ese un mejor default para un test double que devolver silenciosamente un valor cero?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/memory/tools.go` y `agent.go` para el mecanismo real — dos funciones de herramienta, ambas tocando `ctx.State()`, envueltas de la misma forma que las herramientas de `internal/agents/calculator`.
