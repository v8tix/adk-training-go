# Módulo 6: Ejecución Programática: Apps y Runners (Go) ⚙️

## Teoría

### Agente y Runner: Todo Vive en un Solo Config 📦

Construir y manejar un agente de forma programática se reduce en realidad a dos cosas: el agente mismo, y un `*runner.Runner` para ejecutarlo. Cada configuración que necesitarías — almacenamiento de sesiones, artifacts, memoria, plugins, caché de contexto — es simplemente un campo en una sola struct, confirmado leyendo la lista de campos real de `runner.Config`, no inferido:

```go
type Config struct {
    AppName string
    Agent   agent.Agent
    SessionService session.Service
    ArtifactService artifact.Service // optional
    MemoryService   memory.Service   // optional
    PluginConfig    PluginConfig     // optional
    Compaction      *compaction.Config // optional
}
```

`runner.New(cfg)` construye un runner completamente configurado en un solo paso — no hay que armar un objeto de infraestructura separado primero. Lindo, ¿no? 👍

### Manejando Tú Mismo el Stream de Eventos 🌊

`Runner` tiene exactamente dos métodos de ejecución, confirmados vía `go doc`:

```go
func (r *Runner) Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig, opts ...RunOption) iter.Seq2[*session.Event, error]
func (r *Runner) RunLive(...) (...)
```

`Run` te da un iterador sobre los eventos del agente — recorres el iterador y buscas aquel donde `IsFinalResponse()` sea verdadero. Este es el mismo pequeño patrón que todo programa `cmd/` de este repo viene escribiendo desde el módulo 3 (`runEcho`, `analyzeTicket`); el `runOnce` de `cmd/support-analyzer-runner` simplemente lo nombra explícitamente, ya que ponerle nombre al patrón es literalmente lo que enseña este módulo.

### Un Runner, Dos Usuarios Aislados — el Punto Real de Este Módulo 👥

```go
r, _ := runner.NewInMemory("support_analyzer_runner_app", rootAgent)

aliceResult, _ := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
bobResult, _ := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
```

Es el mismo mecanismo de aislamiento por `userID`/`sessionID` que los tests de este repo vienen usando desde el módulo 3 — lo nuevo acá es construir deliberadamente *un solo* `*runner.Runner` y manejar *dos usuarios diferentes* a través de él en el mismo proceso. Esa es exactamente la forma que necesita un backend real (un servidor web manejando requests concurrentes). Confirmado en vivo: el reclamo de facturación de Alice y el problema técnico de Bob obtienen cada uno su propio análisis correcto e independiente desde el mismo runner compartido — cero fugas de estado entre ellos. ✨

### Compartiendo un Agente entre Dos Programas 🤝

Cuando un segundo programa necesita la misma definición de agente, esa definición necesita su propio paquete importable — un paquete de Go solo puede tener un `func main()`, así que no puede vivir dentro de ningún programa `cmd/` directamente. Este módulo saca la definición del Support Analyzer a `internal/agents/supportanalyzer`, y tanto `cmd/support-analyzer` (el punto de entrada CLI/launcher, sin cambios de comportamiento desde el módulo 5) como el nuevo `cmd/support-analyzer-runner` (el entregable real de este módulo) lo importan. Este es el primer paquete `internal/agents/` de este repo — una categoría nueva junto a `internal/infrastructure/`, para lógica de definición de agentes/dominio en vez de adaptadores de sistemas externos: el hogar natural para "el agente mismo" cuando más de un programa necesita ejecutarlo.

### Puntos Clave ✅
- `runner.Config` trae directamente toda la configuración de infraestructura — almacenamiento de sesiones, artifacts, memoria, plugins, compaction — confirmado por su propia lista de campos. `runner.New(cfg)` construye un runner completamente configurado en un solo paso.
- `Run` devuelve un iterador, no una lista plana de eventos — recórrelo y encuentra aquel donde `IsFinalResponse()` sea verdadero, el patrón que este repo usa desde el módulo 3.
- Un `*runner.Runner`, muchos usuarios, aislados por `userID`/`sessionID` — probado en vivo con dos resultados genuinamente diferentes y correctos desde una sola instancia compartida.
- `internal/agents/` es donde vive la definición propia de un agente cuando más de un programa necesita ejecutarlo.

<hr/>

> **¿Vienes de Python?** 🐍 Python nombra tres objetos — `Agent`, `App` (infraestructura: plugins, caché) y `Runner` (el motor de ejecución). Go solo tiene dos: todo lo que el `App` de Python guarda por separado es simplemente un campo del `runner.Config` de Go, así que `runner.New(cfg)` ya *es* "envolver el agente, luego construir un runner" colapsado en un solo paso — no hay ningún tipo `App` que buscar. El wrapper de conveniencia `runner.run_debug()` de Python tampoco tiene equivalente en Go; tú mismo manejas el iterador de `Run`. Y donde el laboratorio de Python agrega un segundo archivo (`main.py`) al mismo directorio del proyecto, Go expresa la misma separación "agente vs. cómo lo invocas" como un límite de paquete, ya que dos `func main()` no pueden compartir un paquete.
