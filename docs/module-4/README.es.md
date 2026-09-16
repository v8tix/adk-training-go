# Módulo 4: Conceptos Centrales del Agente: Análisis Profundo (Go) 🔬

## Teoría

### El "Cerebro" de la Operación

En Go, el plano de un agente es `llmagent.Config`, el mismo tipo que ya usaron los módulos 2-3:

* **`Name`:** un identificador único para tu agente.
* **`Model`:** el `model.LLM` que impulsa al agente — construido vía `internal/infrastructure/llm.BuildModel` en este repo.
* **`Instruction`:** la parte más crítica — el prompt detallado que define la personalidad, objetivos y restricciones del agente.
* **`Description`:** un resumen breve del propósito del agente.

### El Arte de la Instrucción ✍️

Escribe instrucciones como si le explicaras algo a un nuevo empleado que no puede hacer preguntas de seguimiento: sé claro y específico, usa lenguaje simple, da un ejemplo para cualquier cosa tipo clasificación, e itera una vez que veas resultados reales. `cmd/support-analyzer/prompts/support_analyzer_instruction.md` (Markdown, `# Instructions` / `# Constraints`) es el ejemplo de este módulo — enumera los valores exactos permitidos para `category` y `sentiment` en vez de dejarlos abiertos, la misma lección de "sé explícito, no asumas que el modelo va a inferir tus restricciones" que ya enseñó la instrucción del agente eco del módulo 3.

### Dos Campos Más en `llmagent.Config`: Salida Estructurada y Estado

Confirmado leyendo el propio código fuente de `google.golang.org/adk/v2@v2.4.0` (no solo `adk.dev`), `llmagent.Config` tiene dos campos más que vale la pena conocer bien: `OutputSchema`, que fuerza la respuesta final del modelo a una forma JSON que tú defines, y `OutputKey`, que guarda esa respuesta automáticamente en el estado de sesión.

#### 1. Forzando JSON con `OutputSchema`

```go
schema := &genai.Schema{
    Type: genai.TypeObject,
    Properties: map[string]*genai.Schema{
        "category":  {Type: genai.TypeString},
        "sentiment": {Type: genai.TypeString},
        "summary":   {Type: genai.TypeString},
    },
    Required: []string{"category", "sentiment", "summary"},
}

analyzerAgent, err := llmagent.New(llmagent.Config{
    Name:         "support_analyzer_agent",
    Model:        llmModel,
    Instruction:  instruction,
    OutputSchema: schema, // Force JSON output
})
```

El patrón avalado por el SDK aquí (usado en el propio `examples/multiagent/single_turn` del SDK) es un literal `*genai.Schema` escrito a mano, mantenido sincronizado manualmente con cualquier struct de Go que definas para acceso tipado al resultado (`SupportAnalysis` en el código de este módulo) — `genai.InternalTSchema`/`InternalTJsonSchema` existen en el SDK pero su propio comentario de documentación dice "public only for internal purposes... external consumers must not use it", así que no son un atajo aquí. Confirmado en el código fuente (`agent/llmagent/llmagent.go`, un literal `// TODO: add output schema validation and unmarshalling`): `OutputSchema` solo le da forma a la *solicitud* — el SDK nunca valida ni parsea la respuesta JSON del modelo. Tu propio código hace eso, de la misma forma en que la prueba de `cmd/support-analyzer` lo hace con `json.Unmarshal`.

`OutputSchema` y `Tools` se pueden configurar juntos — el agente puede seguir llamando herramientas durante su ciclo de pensamiento; solo la respuesta final queda restringida al esquema.

<hr/>

> **¿Vienes de Python?** 🐍 Python pasa un `BaseModel` de Pydantic y obtiene el esquema *y* la validación/parseo de la misma declaración. Go no tiene un equivalente de auto-derivación — escribes el `*genai.Schema` a mano y lo mantienes sincronizado con tu struct de resultado tú mismo.

#### 2. Pasando Datos con `OutputKey`

```go
llmagent.Config{
    // ...
    OutputKey: "last_ticket_analysis", // Saves output to state["last_ticket_analysis"]
}
```

Confirmado en el código fuente: en el evento de respuesta final del agente, el SDK concatena todas las partes de texto que no son razonamiento (`!Part.Thought`) y escribe el resultado en `event.Actions.StateDelta[OutputKey]`. Esto es directamente inspeccionable desde código Go que maneja el agente vía `runner` — la prueba de `cmd/support-analyzer` lee `event.Actions.StateDelta["last_ticket_analysis"]` directamente del flujo de eventos, sin necesitar una consulta separada al servicio de sesión (aunque `session.InMemoryService()` + `(Service).Get(...)` → `.Session.State()` es el camino equivalente si necesitas inspeccionar el estado desde fuera del bucle de ejecución, por ejemplo después del hecho).

### Un Problema Real de Inferencia Local: No Toda Cuantización Soporta Salida Estructurada ⚠️

Este módulo reveló una limitación real de la infraestructura local, confirmada con pruebas (no asumida): algunas cuantizaciones de la familia de modelos de este curso **no** soportan decodificación restringida por esquema JSON — Ollama mismo devuelve `501 Not Implemented: "structured output is unavailable"` para cualquier solicitud `OutputSchema`/`response_format: json_schema` contra ellas, confirmado tanto en `/v1/chat/completions` como en `/v1/responses` (el endpoint que realmente llama el adaptador `openaimodel` de `internal/infrastructure/llm`).

La solución se queda completamente local, sin necesidad de recurrir a la nube: `qwen3.8:27b`, una cuantización GGUF `Q4_K_M` de la misma familia de modelos, **sí** lo soporta — confirmado con la solicitud idéntica contra los endpoints idénticos, `200 OK` con JSON válido conforme al esquema ambas veces, y confirmado funcionando también para módulos de texto plano (echo-agent, verify-setup). Por eso, **`qwen3.8:27b` ahora es el `OLLAMA_MODEL` compartido por defecto de este repo** (`internal/infrastructure/llm/config.go`) — sin necesitar sobrescritura por módulo, ni aquí ni en ningún módulo anterior. 🎉

### Puntos clave ✅
- `llmagent.Config{OutputSchema, OutputKey}` te permite forzar una respuesta JSON conforme al esquema y guardarla automáticamente en el estado de sesión — confirmado leyendo el propio código fuente del SDK, no solo su documentación.
- No hay auto-derivación de esquemas en Go: escribes a mano un `*genai.Schema` y lo mantienes sincronizado con tu struct de resultado tú mismo.
- El SDK le da forma a la *solicitud* vía `OutputSchema` pero nunca valida ni parsea la *respuesta* — tu propio código hace eso.
- `OutputKey` escribe en `event.Actions.StateDelta`, directamente inspeccionable desde una prueba manejada por `runner`, sin necesitar un viaje de ida y vuelta al servicio de sesión.
- No toda cuantización de modelo local soporta salida estructurada — verifícalo empíricamente por modelo/cuantización, no asumas que "simplemente funciona" porque la generación de texto plano funcionó.
