# Módulo 8: Introducción a las Herramientas (Go) 🔧

## Teoría

### De Hablar a Actuar 💪

Todo agente de los módulos anteriores solo podía razonar sobre lo que su modelo ya sabía de su entrenamiento. Este módulo le da a un agente su primera forma de romper esa caja: una **Herramienta (Tool)** — una capacidad que un agente puede invocar para hacer algo más allá de generar texto, como buscar en la web información que ni siquiera existía cuando el modelo fue entrenado.

### `tool.Tool` y `llmagent.Config.Tools` 🛠️

En el SDK de Go de ADK, una herramienta es cualquier cosa que implemente `tool.Tool` (`Name()`, `Description()`, `IsLongRunning()`). Adjuntar una a un agente es exactamente un campo en el config que ya vienes usando desde el módulo 2:

```go
llmagent.New(llmagent.Config{
    Name:        "researcher_agent",
    Model:       llmModel,
    Instruction: instruction,
    Tools:       []tool.Tool{geminitool.GoogleSearch{}},
})
```

`geminitool.GoogleSearch{}` es una **herramienta integrada (built-in)**: a diferencia de una herramienta de función personalizada (próximo módulo), corre completamente dentro del modelo Gemini mismo — el framework de ADK nunca llama a ningún código local para ella. Su método `ProcessRequest` simplemente adjunta `&genai.Tool{GoogleSearch: &genai.GoogleSearch{}}` al request saliente; Gemini decide cuándo buscar y integra los resultados en su propio razonamiento antes de responder.

### Una Brecha Real y Confirmada en la Inferencia Local — la Misma Categoría que la del Módulo 7 🔬

Este módulo descubrió la misma clase de brecha que el módulo 7 encontró para imágenes, en un camino de código diferente del SDK:

1. **`google_search` es una herramienta integrada nativa de Gemini, no una función que ADK llama localmente.** Confirmado vía código fuente: el `ProcessRequest` de `tool/geminitool/google_search.go` solo define `genai.Tool.GoogleSearch` — no hay ningún método `Run` local para ejecutar.
2. **El cliente Go de este repo para el backend local de Ollama la rechaza directamente.** El `ensureFunctionToolOnly` de `model/openaimodel/tools.go` explícitamente revisa y rechaza `GoogleSearch` (junto con cualquier otra herramienta integrada no-función: `Retrieval`, `GoogleMaps`, `CodeExecution`, etc.) con `"openai: non-function tools are not supported (tool %d)"`. Confirmado en vivo: construir un `researcher_agent` con esta herramienta contra el backend de Ollama y ejecutarlo falla inmediatamente con exactamente ese error, antes de que salga ninguna llamada de red.

**La afirmación precisa y correcta es: el cliente de este SDK para modelos locales no puede enviar ninguna herramienta integrada que no sea función — no que Ollama o el modelo subyacente no tengan capacidad de búsqueda para nada.** Por eso `cmd/researcher` requiere `MODEL_TYPE=gemini`, hardcodeado en su propio `main()`, igual que `cmd/visual-catalog` en el módulo 7.

### `google_search` Tampoco Necesita Configuración Nueva ✨

Confirmado en vivo: el mismo camino de `GOOGLE_AI_STUDIO_API_KEY` que todo módulo desde el módulo 2 ya usa alcanza también para `google_search` — sin necesitar configuración separada de proyecto/ubicación.

```
$ go run ./temp/module-8/probe2   # a researcher_agent, MODEL_TYPE=gemini, existing GOOGLE_AI_STUDIO_API_KEY
Using model: gemini-3.5-flash
GroundingMetadata present: true
WebSearchQueries: [...]
Answer: <a real, current, grounded answer>
```

Sin `GOOGLE_GENAI_USE_VERTEXAI`, `GOOGLE_CLOUD_PROJECT`, ni `GOOGLE_CLOUD_LOCATION` definidos en ningún lado — el mismo camino de `GOOGLE_AI_STUDIO_API_KEY` que todo módulo desde el módulo 2 ya usa resultó ser suficiente. 🎉

### Probando que la Herramienta Realmente se Disparó: `event.GroundingMetadata` 🔍

`session.Event` (devuelto por cada llamada a `runner.Run`) incluye `model.LLMResponse`, que trae `GroundingMetadata *genai.GroundingMetadata` directamente — no nulo exactamente cuando ocurrió una búsqueda real. Eso te da una forma automatizada y estructural de verificar que una herramienta realmente se disparó, en vez de solo poder revisarlo a ojo en la vista de Trace del Dev UI (que sigue ahí, y sigue siendo útil — mira el laboratorio).

### Puntos Clave ✅
- Una **herramienta integrada** como `google_search` corre dentro del modelo mismo; una **herramienta de función personalizada** (próximo módulo) es tu propio código que ADK llama localmente. `llmagent.Config.Tools []tool.Tool` es el punto de conexión para ambas.
- El cliente Go del backend local de Ollama no puede enviar ninguna herramienta integrada que no sea función (confirmado vía la revisión exacta del código fuente y un error real reproducido) — una limitación real y precisamente delimitada del SDK, no una afirmación sobre las capacidades propias de Ollama o del modelo.
- El camino simple de `GOOGLE_AI_STUDIO_API_KEY` ya soporta `google_search`, confirmado en vivo — sin necesitar configuración nueva.
- `event.GroundingMetadata` te da una forma real y automatizable de verificar que una herramienta integrada se disparó, en vez de depender solo de la inspección manual de la vista de Trace.

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python pide una configuración completa de Vertex AI para `google_search` (`GOOGLE_GENAI_USE_VERTEXAI`, un project ID, una location) porque la herramienta "requiere una configuración de Agent Platform". El camino más simple de este repo con `GOOGLE_AI_STUDIO_API_KEY` la maneja perfectamente bien, confirmado en vivo — la segunda vez que este curso encuentra que un requisito de Vertex AI declarado por Python no es necesario acá (el módulo 7 encontró lo mismo para visión). Y donde el laboratorio de Python verifica que la herramienta se disparó leyendo manualmente la vista de Trace del Dev UI, `event.GroundingMetadata` te permite revisar eso automáticamente en cambio.
