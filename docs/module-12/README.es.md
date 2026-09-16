# Módulo 12: Herramientas Prediseñadas y Grounding (Go) 🌐

## Teoría

### Una Herramienta Que Corre Adentro del Modelo

Cada herramienta que has construido hasta ahora — las funciones aritméticas de `internal/agents/calculator`, la búsqueda de moneda de `internal/agents/marketanalyst` — es código que el framework de ADK llama en tu nombre, localmente, en tu propio proceso. Una **herramienta prediseñada** invierte eso por completo: es una capacidad que el modelo mismo corre, dentro de la propia infraestructura de Google, sin ejecución de código local en absoluto. `google_search` es una: dale a un modelo Gemini 2.0+ esta herramienta, y puede decidir, por su cuenta, buscar en la web en vivo antes de responder. 🔍

Agregarla es un cambio de una línea en el `Tools` de un agente:

```go
import (
    "google.golang.org/adk/v2/agent/llmagent"
    "google.golang.org/adk/v2/tool"
    "google.golang.org/adk/v2/tool/geminitool"
)

rootAgent, err := llmagent.New(llmagent.Config{
    Name:        "research_agent",
    Model:       llmModel,
    Instruction: "Use google_search to find current information, then summarize it.",
    Tools:       []tool.Tool{geminitool.GoogleSearch{}},
})
```

`geminitool.GoogleSearch{}` es un struct de cero campos que satisface la misma interfaz `tool.Tool` que toda herramienta personalizada (`Name`, `Description`, `ProcessRequest`) — solo agrega una entrada `genai.Tool{GoogleSearch: &genai.GoogleSearch{}}` a la solicitud en vez de un esquema JSON de función. `internal/agents/researcher` (módulo 8) ya construye un agente alrededor de exactamente esta herramienta; este módulo va un paso más allá: ¿qué pasa cuando *también* quieres tus propias herramientas personalizadas en la mezcla?

### La Restricción Real de la API de Gemini

Intenta agregar una herramienta de función personalizada a ese mismo agente, y la solicitud misma falla — no en tiempo de construcción, sino en el momento en que el modelo realmente corre:

```
400 INVALID_ARGUMENT: Please enable tool_config.include_server_side_tool_invocations
to use Built-in tools with Function calling.
```

Confirmado en vivo en este módulo: esta es una restricción genuina de la API de Gemini, no algo que el framework de ADK — Go o Python — imponga o pueda relajar por su cuenta. Por defecto, una solicitud puede llevar `google_search` *o* herramientas declaradas por función, nunca ambas. 🚫

### Levantando la Restricción, o Solucionándola de Otra Forma

El mensaje de error de arriba nombra directamente su propia solución. Activar `genai.ToolConfig.IncludeServerSideToolInvocations` realmente levanta la restricción — confirmado en vivo: un agente, llevando tanto `geminitool.GoogleSearch{}` como dos herramientas de función personalizadas, buscó exitosamente en la web (`GroundingMetadata` real, tres chunks de grounding) *y* llamó a una herramienta personalizada, en la misma conversación. 🎉

```go
includeServerSide := true
combinedAgent, err := llmagent.New(llmagent.Config{
    Name:        "combined_research_agent",
    Model:       llmModel,
    Instruction: "Search for the topic, then extract and format the findings.",
    Tools:       []tool.Tool{geminitool.GoogleSearch{}, extractFactsTool, formatNotesTool},
    GenerateContentConfig: &genai.GenerateContentConfig{
        ToolConfig: &genai.ToolConfig{
            IncludeServerSideToolInvocations: &includeServerSide,
        },
    },
})
```

`llmagent.Config.GenerateContentConfig` es un passthrough directo a `genai.GenerateContentConfig` — el mismo struct exacto que le darías al cliente de Gemini subyacente — así que esto no necesita ningún cableado extra más allá de activar un campo. Mira `internal/agents/researchassistant.BuildCombinedAgent` para la versión funcionando, y `TestCombinedAgent_UsesSearchAndCustomTool_Gemini` para la prueba en vivo.

La otra opción: **composición secuencial**. Deja la restricción como está, usa dos agentes separados en su lugar — uno solo con `google_search`, uno solo con tus herramientas personalizadas — llamando al primero, y luego entregando su salida de texto plano al segundo como entrada. Esta sigue siendo una elección totalmente legítima incluso ahora que el enfoque combinado funciona: mantiene la superficie de herramientas de cada agente mínima y su propia responsabilidad, y es lo que construye a mano el laboratorio de este módulo:

```go
findings, err := runAgent(ctx, researchAgent, "research_app", "Research this topic: "+topic)
report, err := runAgent(ctx, formatterAgent, "formatter_app", "Topic: "+topic+"\n\nFindings: "+findings)
```

Dos llamadas a `runner.Run`, una por agente, en tu propio código — la misma forma de ejecución programática que ya usaron los módulos 6 y 10 por otras razones.

### `google_maps_grounding`: Una Construcción Real, No un Laboratorio Construido

Gemini también tiene una herramienta prediseñada de grounding por ubicación, `google_maps_grounding`, para preguntas como "qué hay cerca de mí" o "cómo llego ahí". `geminitool` no la trae como su propio tipo nombrado como sí lo hace con `GoogleSearch`, pero el comentario de documentación de su propio paquete dice exactamente cómo agregar cualquier herramienta nativa de Gemini: `geminitool.New(name, description, &genai.Tool{...})`. `genai.GoogleMaps` es un tipo real y presente, así que esto genuinamente compila:

```go
mapsGrounding := geminitool.New(
    "google_maps_grounding",
    "Answers location-based questions using Google Maps.",
    &genai.Tool{GoogleMaps: &genai.GoogleMaps{}},
)
```

Este módulo no la construye ni la prueba, sin embargo, coincidiendo con el propio alcance de este curso aquí — `google_maps_grounding` necesita la API de Vertex AI, y la configuración de este repo solo configura una key plana de AI Studio.

### Puntos Clave ✅
- Una herramienta prediseñada como `google_search` corre dentro del entorno propio del modelo — sin ejecución de código local, agregada con una línea: `Tools: []tool.Tool{geminitool.GoogleSearch{}}`.
- La API de Gemini rechaza mezclar una herramienta prediseñada con herramientas de función personalizadas por defecto — confirmado en vivo con el error real `400 INVALID_ARGUMENT`.
- Activar `genai.ToolConfig.IncludeServerSideToolInvocations` en `llmagent.Config.GenerateContentConfig` levanta esa restricción, dejando que un agente use genuinamente ambos tipos de herramienta juntos — confirmado en vivo con metadata de grounding real y una llamada real a herramienta personalizada en la misma conversación.
- La composición secuencial — la salida de un agente solo-búsqueda alimentando a un agente solo-herramientas-personalizadas, vía dos llamadas separadas a `runner.Run` — sigue siendo una alternativa legítima y de alcance más simple incluso ahora que el enfoque combinado funciona.
- `google_maps_grounding` es real (`geminitool.New` + `genai.GoogleMaps`) pero necesita Vertex AI, fuera del alcance de este curso.
- No existe ningún tipo `ManagedAgent` en ninguna parte del código fuente fijado de `google.golang.org/adk/v2 v2.4.0` — un vacío de alcance confirmado en esta versión del SDK, no un patrón para construir todavía.

<hr/>

> **¿Vienes de Python?** 🐍 El propio módulo 12 de Python describe la restricción de herramientas mezcladas como algo absoluto, solucionado solo dividiendo en dos agentes — nunca menciona `include_server_side_tool_invocations`. Esa bandera existe a nivel de la API de Gemini, no solo en este SDK de Go, así que el mismo enfoque de agente combinado debería ser alcanzable desde Python también; simplemente no es parte del currículum de Python de este curso. Trata el patrón de dos agentes como el que enseña este curso en ambos lados, y el agente combinado como un bonus real que este módulo de Go expone. Por separado, el ADK de Python también tiene un `ManagedAgent` en preview para los agentes propios de Google alojados en servidor — esta versión del SDK de Go todavía no tiene equivalente, según los Puntos Clave de arriba.
