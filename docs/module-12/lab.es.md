# Laboratorio 12: Construyendo un Asistente de Investigación con Búsqueda Web (Go) 🔎

## Objetivo

Construye un asistente de investigación que se fundamenta en resultados web en vivo, y luego le pasa esos resultados a un segundo agente que los formatea en un reporte — el patrón de composición secuencial del README de este módulo, construido de punta a punta.

### Prerrequisitos

Este laboratorio necesita una `GOOGLE_AI_STUDIO_API_KEY` real en tu `.env` — `google_search` solo funciona en modelos Gemini 2.0+, y el backend local de Ollama la rechaza de plano antes de siquiera llegar a la red (confirmado en el módulo 8). No hay un camino solo-local para este laboratorio, ¡lo siento! 😅

### Paso 1: Los Dos Agentes

`internal/agents/researchassistant/agent.go` define dos agentes:

```go
func BuildResearchAgent(llmModel model.LLM) (agent.Agent, error) {
    // Only geminitool.GoogleSearch{} in Tools.
}

func BuildFormatterAgent(llmModel model.LLM) (agent.Agent, error) {
    // Only extract_key_facts and format_research_notes in Tools — never google_search.
}
```

Las dos herramientas personalizadas que llama el agente formateador viven en `tools.go`:

- `extractKeyFacts(ctx, ExtractKeyFactsArgs{Text, NumFacts}) (ExtractKeyFactsResult, error)` — divide `Text` por `.`, se queda con oraciones más largas de 10 caracteres, hasta `NumFacts` de ellas.
- `formatResearchNotes(ctx, FormatResearchNotesArgs{Topic, Findings}) (FormatResearchNotesResult, error)` — arma un reporte estilo Markdown con una marca de tiempo generada.

Lee ambos constructores de agentes y ambas funciones de herramienta antes de seguir — este laboratorio trata de cómo se orquestan, no de escribir nueva lógica de herramienta.

### Paso 2: Corre el Pipeline 🏃

`cmd/research-assistant/main.go`'s `runResearchPipeline` hace la orquestación:

```go
findings, err := runAgent(ctx, researchAgent, "research_app", "Research this topic: "+topic)
report, err := runAgent(ctx, formatterAgent, "formatter_app", "Topic: "+topic+"\n\nFindings: "+findings)
```

`runAgent` es un pequeño helper alrededor de `runner.NewInMemory` + `runner.Run`, devolviendo el texto final sin pensamiento del agente — la misma forma de ejecución programática de los módulos 6 y 10, solo que llamada dos veces con dos agentes diferentes.

Córrelo:

```bash
go run ./cmd/research-assistant "the latest AI developments from Google"
```

Salida real y confirmada de este comando exacto (abreviada — los hallazgos de investigación continúan con tres desarrollos más en viñetas, y la sección de Hallazgos del reporte tiene tres hechos más):

```
🔎 research-assistant using gemini-3.5-flash
--- RESEARCH FINDINGS ---
In 2026, Google's artificial intelligence developments reflect a major
industry-wide shift from passive chat queries toward proactive "agentic AI"—
systems designed to execute complex, multi-step workflows autonomously.

Major developments include:

*   **The Gemini 3.5 Family & Next-Gen Models:** Google launched the
    **Gemini 3.5** model line, with **Gemini 3.5 Flash** (and the
    later-released **Gemini 3.8 Flash**) optimized for fast, autonomous
    execution loops. [...]
*   **Gemini Omni:** Developed by Google DeepMind, **Gemini Omni** is a
    unified multimodal model capable of generating high-quality video and
    native audio from any combination of inputs. [...]
[...]

--- FINAL REPORT ---
# Research Report: the latest AI developments from Google
Generated: 2026-09-15 09:27:29

## Findings
- In 2026, Google's artificial intelligence developments reflect a major
  industry-wide shift from passive chat queries toward proactive "agentic
  AI"—systems designed to execute complex, multi-step workflows autonomously.
- Google launched the Gemini 3.5 model line, with Gemini 3.5 Flash and
  Gemini 3.8 Flash optimized for fast, autonomous execution loops, with
  rumors of a Gemini 4 model in late 2026.
- Developed by Google DeepMind, Gemini Omni is a unified multimodal model
  capable of generating high-quality video and native audio, including
  progressive video editing.
[...]
```

Fíjate: los hallazgos del agente de investigación reflejan eventos reales y actuales más allá de cualquier corte de entrenamiento — prueba de que `google_search` realmente corrió — y el agente formateador nunca tocó la web en absoluto, solo el texto de hallazgos que se le entregó. ¡Bonita separación de responsabilidades! 🎯

### Paso 3 (Bonus): Un Agente, Ambos Tipos de Herramienta 🎁

`BuildCombinedAgent` en el mismo paquete muestra la alternativa que cubre el README de este módulo: en vez de dos agentes, un agente con `IncludeServerSideToolInvocations` activado puede usar `google_search` y tus herramientas personalizadas juntas. Lee `TestCombinedAgent_UsesSearchAndCustomTool_Gemini` en `agent_test.go` para ver cómo se prueba eso — chequea tanto `GroundingMetadata` real *como* una `FunctionResponse` real de `format_research_notes` en la misma conversación.

### Solución de Problemas

¿Te trabaste? Mira [troubleshooting.md](./troubleshooting.md).

### Resumen del Laboratorio 🎉

Construiste un pipeline de investigación de dos agentes usando la herramienta prediseñada `google_search` del ADK, aprendiste exactamente por qué la API de Gemini rechaza mezclarla con herramientas personalizadas por defecto, y viste ambas formas de solucionarlo: dividir en dos agentes, o activar una bandera para combinarlos en uno. ¡Buen trabajo!

### Preguntas de Autorreflexión 🤔
- ¿Por qué correr `google_search` "adentro del modelo" lo hace un tipo de herramienta fundamentalmente diferente a `extract_key_facts`, que corre en tu propio proceso Go?
- `TestMixedTools_WithoutServerSideFlag_Fails_Gemini` construye deliberadamente un agente inválido para probar que la restricción es real. ¿Por qué un test que espera fallar es igual de valioso que uno que espera éxito?
- Ahora que sabes que `IncludeServerSideToolInvocations` existe, ¿cuándo seguirías eligiendo la composición secuencial (dos agentes) por sobre el enfoque combinado de un solo agente?

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python solo construye la versión de composición secuencial — `formatter_agent` se deja como `TODO` para que lo completes tú, y luego el helper `run_agent` de `main.py` hace la misma orquestación de dos llamadas que hace `runAgent` acá. Este laboratorio de Go agrega el Paso 3 como contenido bonus sin equivalente en Python en este curso.
