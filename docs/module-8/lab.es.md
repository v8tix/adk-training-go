# Laboratorio 8: Construyendo un Agente "Investigador" con Google Search (Go) 🔎

## Objetivo

Construye un agente que pueda buscar en la web para responder preguntas sobre eventos actuales, usando la herramienta integrada `google_search` de ADK.

## Tareas del Laboratorio

### 1. Lee `internal/agents/researcher/agent.go` 📖

La misma forma que `internal/agents/visualcatalog` (módulo 7) — `BuildRootAgent(llmModel)` resuelve su propio prompt internamente — con un agregado: `Tools: []tool.Tool{geminitool.GoogleSearch{}}` en el `llmagent.Config`. Ese único campo es todo el cableado necesario para una herramienta integrada. ¡Eso es todo! 🙌

### 2. Lee `internal/agents/researcher/prompts/researcher_instruction.md` 📖

Nota que no solo dice "tienes una herramienta de búsqueda" — dice explícitamente *cuándo* usarla (eventos actuales, datos en vivo, cualquier cosa sensible al tiempo) y cuándo no (cualquier cosa respondible con conocimiento general). Esto es deliberado: un agente con una herramienta pero sin guía sobre cuándo usarla puede buscar innecesariamente, o no buscar cuando debería.

### 3. Ejecútalo — modo consola ▶️

```bash
go run ./cmd/researcher console
```

(Requiere `GOOGLE_AI_STUDIO_API_KEY` en tu entorno o `.env` — mira `.env.example`.)

Salida real y confirmada de este comando exacto:

```
🔎 researcher using gemini-3.5-flash

User -> What is the current weather in Tokyo right now?
Agent -> As of the evening of Monday, September 14, 2026, in Tokyo, the current weather conditions are:

* **Temperature:** Approximately 83°F to 84°F (28°C)
* **Conditions:** Mostly cloudy with passing clouds
...
```

Una respuesta real, actual, y fundamentada en datos — algo que los datos de entrenamiento propios de un modelo jamás podrían contener. 🌍

### 4. Ejecútalo — modo Dev UI, e inspecciona la vista de Trace 🕵️

> **Elección de puerto:** este laboratorio usa `9091` en vez del puerto por defecto del launcher de ADK, `8080` — `8080` suele estar ocupado por otras herramientas de desarrollo local (mira el laboratorio del módulo 3 para una colisión real confirmada en la máquina de esta sesión, cortesía del proxy propio de Docker Desktop). Si `9091` también está ocupado en tu máquina, revisa con `lsof -i :9091` y elige cualquier otro puerto libre — solo ajusta la URL de abajo **y** el flag `-api_server_address` para que coincida.
>
> **Un gotcha real y confirmado: `--port` solo no alcanza.** ⚠️ El frontend del Dev UI aprende dónde llamar a la API desde un flag *separado*, el propio `-api_server_address` de `webui`, que por defecto es el hardcodeado `http://localhost:8080/api` sin importar `--port` — confirmado en vivo (mira el README del módulo 3 para la historia completa). Por eso el comando de abajo lo define explícitamente.

```bash
go run ./cmd/researcher web --port 9091 webui -api_server_address http://localhost:9091/api api
```

Abre `http://localhost:9091/`, haz la misma pregunta sobre el clima, y luego abre la vista de **Trace** para ese turno. Deberías ver la llamada de función `google_search` y su resultado como un paso distinto antes de la respuesta final del modelo.

Confirmado en vivo en este repo: `curl http://localhost:9091/api/list-apps` devuelve `["researcher_agent"]`, y el Dev UI mismo (`http://localhost:9091/`, redirigiendo a `/ui/`) devuelve `200`.

### 5. Mira la limitación del backend local, a propósito 💥

Abre el comentario de documentación del paquete en `internal/agents/researcher/agent.go` y el comentario de documentación de `cmd/researcher/main.go` — ambos explican que `MODEL_TYPE` está forzado a `gemini` en el código. ¿Curioso por qué? Intenta construir un `researcher_agent` contra el backend de Ollama tú mismo (mira `temp/module-8/probe` si tienes acceso al árbol de trabajo de este repo, o simplemente lee su salida confirmada abajo) — te vas a topar con el mismo error real del que trata este módulo:

```
RUN ERROR: openai: non-function tools are not supported (tool 0)
```

## Preguntas de Autorreflexión 🤔
- ¿Por qué es importante instruir explícitamente al agente sobre *cuándo* usar la herramienta `google_search`? ¿Qué podría pasar si simplemente le dieras la herramienta sin instrucciones? (Mira la sección `# Constraints` de `researcher_instruction.md` para la respuesta de este repo.)
- La versión en Python de este laboratorio afirma que `google_search` requiere una configuración de Agent Platform (Vertex AI). Este repo encontró que esa afirmación no se sostiene para su propia configuración — ¿cuál es el riesgo de tomar la documentación de un framework al pie de la letra en vez de probar una afirmación así directamente?
- ¿Cómo cambia fundamentalmente darle a un agente acceso a información en tiempo real el tipo de problemas que puede resolver, comparado con un agente que solo depende de su conocimiento interno?
- `event.GroundingMetadata` le permite al test de este repo verificar que la herramienta se disparó sin que un humano abra la vista de Trace. ¿Cuál es una desventaja de depender solo de una revisión estructural automatizada así, en vez de mirar la vista de Trace tú mismo alguna vez?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/researcher/agent.go` y `cmd/researcher/main.go` para el mecanismo real — adjuntar una herramienta integrada es un campo en `llmagent.Config`, y el resto es el mismo patrón de launcher que el módulo 5 ya estableció.
