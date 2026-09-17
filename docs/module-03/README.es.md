# Módulo 3: Tu Primer Agente: El Agente "Eco" (Go) 🦜

## Teoría

### El núcleo de un Agente de ADK

En su esencia, un Agente de ADK es un plano que le dice a un Modelo de Lenguaje Grande (LLM) cómo comportarse. En Go, ese plano es `llmagent.Config`:

* **`Name`:** un identificador único para tu agente.
* **`Model`:** el `model.LLM` que actúa como el "cerebro" del agente — construido vía `internal/infrastructure/llm.BuildModel` en este repo (Ollama local por defecto, Gemini opcional).
* **`Instruction`:** la parte más crítica. Este es el prompt detallado que define la personalidad, objetivos y restricciones del agente.
* **`Description`:** un resumen breve y legible del propósito del agente.

### Definiendo un Agente en Código

Construyes un agente con `llmagent.New(llmagent.Config{...})` — código Go plano, sin archivo de configuración que escribir o parsear:

```go
rootAgent, err := llmagent.New(llmagent.Config{
    Name:        "echo_agent",
    Model:       llmModel,
    Description: "An agent that repeats the user's input.",
    Instruction: instruction,
})
```

`instruction` no viene de una constante de string en Go — las instrucciones se vuelven largas y repetitivas rápido (mira `prompts/echo_instruction.md`), y una constante multilínea con backticks es incómoda de leer y editar. En vez de eso, viene de `internal/infrastructure/prompts`, un pequeño caché compartido en el que cada programa `cmd/` puede registrar sus propios prompts:

```go
//go:embed prompts/*.md
var promptFS embed.FS

func init() {
    promptFiles, _ := fs.Sub(promptFS, "prompts") // requerido — ver abajo
    prompts.Register("echo-agent", promptFiles, ".md")
}

// en main():
instruction, err := prompts.Get("echo-agent/echo_instruction")
```

`//go:embed` tiene alcance de directorio — una directiva solo puede alcanzar archivos en o debajo del directorio del archivo que la declara, así que el embed en sí nunca puede moverse al paquete compartido; cada programa `cmd/` sigue embebiendo su propio `prompts/*.md` local. Lo que sí se comparte es el caché y la búsqueda, indexados por `"<namespace>/<name>"` (el namespace es el nombre del propio programa, así que dos programas con archivos de prompt del mismo nombre no colisionan). `fs.Sub(promptFS, "prompts")` es obligatorio, no opcional: `//go:embed prompts/*.md` mantiene el prefijo `"prompts/"` dentro del `embed.FS` resultante, así que saltarte `fs.Sub` registra todo silenciosamente un nivel más profundo de lo esperado.

### Ejecutando el Agente: `cmd/launcher` 🚀

Definir el agente es solo el primer paso. El `cmd/echo-agent` de este repo envuelve `rootAgent` con `agent.NewSingleLoader(rootAgent)` y se lo entrega a un launcher:

```go
config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
l.Execute(ctx, config, os.Args[1:])
```

Esto te da dos modos de ejecución gratis, confirmados corriendo ambos en esta sesión:

* **`go run ./cmd/echo-agent web --port 9091 webui -api_server_address http://localhost:9091/api api`** — inicia una Dev UI local en `http://localhost:9091/ui/` (nota la posición de las flags: las propias flags de `web` van justo después de `web`, y luego cada palabra clave de sub-launcher seguida de sus propias flags; el laboratorio usa `9091` en vez del puerto por defecto del launcher, `8080`, que suele estar ocupado por otras herramientas de desarrollo local — mira el laboratorio para ver una colisión real confirmada). **Tanto `webui` como `api` son obligatorios juntos** — confirmado en vivo en el módulo 5: el frontend de la Dev UI llama a la API REST (`api`) para todo lo que no sea servir su propia página estática, así que `webui` solo inicia una UI que da 404 en cuanto intentas usarla de verdad. **`webui`'s `-api_server_address` debe configurarse explícitamente cada vez que `--port` no sea 8080** — es una flag separada que le dice al frontend del navegador dónde llamar a la API, con su propio valor por defecto hardcodeado `http://localhost:8080/api` independiente de `--port`; si te la saltas, la Dev UI sigue intentando el puerto 8080 sin importar en qué puerto realmente corre el servidor, confirmado en vivo vía `/ui/assets/config/runtime-config.json`.
* **`go run ./cmd/echo-agent console`** — un modo de chat CLI sin navegador, perfecto para cuando no quieres tener un navegador abierto. 💻

**Detalle importante, confirmado al correrlo de verdad:** tanto la UI de `console` como la de `web` muestran la respuesta cruda del modelo, incluyendo su razonamiento paso a paso (chain-of-thought), cuando usas el modelo con capacidad de razonamiento que es el default de este curso — no aplican el filtrado de `Thought` que el propio código de este repo hace en otras partes (`firstAnswerText`). Si ves texto de razonamiento visible antes de la respuesta ecoada en la UI, eso es el renderizador propio del SDK, no un bug en el código de este módulo.

No hay un equivalente confirmado en Go de la pestaña "Trace" de la Dev UI de Python específicamente — el modo `web` del launcher sí integra OpenTelemetry (el paquete `telemetry`), lo cual sugiere fuertemente que existe *alguna* vista de observabilidad, pero esto no se verificó directamente en esta sesión.

### Empezando un Nuevo Programa de Agente

No hay un comando de scaffolding para generar un nuevo proyecto de agente — simplemente escribes `main.go` directamente, de la misma forma en que `cmd/verify-setup` y `cmd/echo-agent` de este repo ya lo hacen. `cmd/adkgo`, el CLI instalable del propio SDK, es para *despliegue* (Cloud Run, Agent Engine), no para generar proyectos.

### Puntos clave ✅
- Un agente de ADK en Go se define completamente en código: `llmagent.Config{Name, Model, Instruction, Description}`.
- `cmd/launcher` te da los modos `web` (Dev UI) y `console` (chat CLI) gratis, una vez que envuelves un agente en `agent.NewSingleLoader`.
- Un nuevo programa de agente empieza como un `main.go` escrito a mano — no hay comando generador, así que lo construyes a partir de los patrones que ya existen en los programas `cmd/` de este repo.
- El razonamiento crudo de un modelo con capacidad de pensar se filtra a través del renderizado propio de la UI del launcher; fíltralo tú mismo (como hacen las pruebas de este repo) cada vez que necesites solo la respuesta, no mostrada a un humano.

<hr/>

> **¿Vienes de Python?** 🐍 El curso de Python ofrece una segunda forma de definir un agente (un archivo de configuración YAML) y un asistente de scaffolding (`uv run adk create <name>`) que genera un esqueleto de proyecto. El SDK de Go no tiene ninguno de los dos — `llmagent.Config` en código es la única forma de definir un agente, y cada programa `cmd/` de este repo comenzó como un `main.go` escrito a mano.
