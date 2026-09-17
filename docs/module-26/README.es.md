# Módulo 26: Callbacks y Guardarraíles — Seguridad y Monitoreo del Agente (Go) 🚦🧯

## Teoría

### Seis Ganchos Hacia la Propia Ejecución de un Agente

Un **callback** intercepta una etapa específica de la ejecución de un agente específico — antes o después de que el agente corra, antes o después de que llame al modelo, antes o después de que llame a una herramienta. `llmagent.Config` expone las seis como simples espacios de función, cada una un slice plural (puedes registrar más de un callback por etapa — Go soporta esto nativamente, a diferencia de un diseño de un solo callback por espacio):

```go
llmagent.Config{
    BeforeAgentCallbacks: []agent.BeforeAgentCallback{cache.beforeAgentCallback},
    AfterAgentCallbacks:  []agent.AfterAgentCallback{cache.afterAgentCallback},
    BeforeModelCallbacks: []llmagent.BeforeModelCallback{beforeModelCallback},
    AfterModelCallbacks:  []llmagent.AfterModelCallback{afterModelCallback},
    BeforeToolCallbacks:  []llmagent.BeforeToolCallback{beforeToolCallback},
    AfterToolCallbacks:   []llmagent.AfterToolCallback{afterToolCallback},
}
```

### Devolver un Valor No Nulo Significa "Usa Esto en Su Lugar"

Cada callback "before" puede cortocircuitar el paso que resguarda devolviendo un resultado no nulo: `BeforeAgentCallback` devolviendo un `*genai.Content` salta por completo la propia ejecución del agente (el mecanismo de caché que construye este módulo); `BeforeModelCallback` devolviendo un `*model.LLMResponse` salta la llamada real al modelo (el guardarraíl de entrada); `BeforeToolCallback` devolviendo un mapa de resultado salta la llamada a la herramienta (validación de argumentos). Los callbacks "after" funcionan igual pero al revés — un retorno no nulo reemplaza lo que ya ocurrió, dejando que `AfterModelCallback` redacte una respuesta real del modelo y que `AfterToolCallback` redacte un resultado real de herramienta después del hecho.

### Callbacks vs. Plugins: Dos Alcances Distintos, Ambos Reales en Go

Los módulos 25 y 25.5 ya construyeron dos Plugins reales (`AlertingPlugin`, `PIIGuardrailPlugin`) — ambos registrados una sola vez a nivel del Runner/launcher vía `PluginConfig`, observando o interceptando *cada* agente al que ese plugin esté conectado. Los callbacks son el alcance opuesto: registrados directamente en el propio `llmagent.Config` de *un* agente, como parte de la propia lógica de ese agente. Si quieres un guardarraíl protegiendo a cada agente de una app, recurre a un Plugin (el propio `PIIGuardrailPlugin` del módulo 25.5 es un ejemplo real). Si quieres cambiar cómo *este agente específico* cachea, valida, o filtra, un callback — conectado directamente en su propia configuración, aquí mismo — es la herramienta correcta.

### Una Restricción Real Encontrada al Construir este Módulo: los Contextos de Callback No Soportan `Session()`

El propio laboratorio de Python lee `callback_context.session.events` directamente para encontrar la última respuesta real de un agente, recorrida en reversa. La primera versión del propio `afterAgentCallback` de este módulo intentó lo mismo en Go — y entró en pánico al correr de verdad: `agent.Context.Session()`, cuando el contexto es un contexto de callback, está deliberadamente restringido. Confirmado leyendo el propio código fuente del SDK (`callback_context_wrapper.go`): registra `"Session() is not supported for callback context"` y devuelve `nil`, lo cual causa pánico en el momento en que algo llama a un método sobre él.

El mecanismo real y respaldado de Go es distinto, y podría decirse que más limpio: `llmagent.Config.OutputKey`. Su propio comentario de documentación nombra exactamente este caso de uso — *"Extracts agent reply for later use, such as in tools, callbacks, etc."* (Extrae la respuesta del agente para uso posterior, como en herramientas, callbacks, etc.) Establécelo una vez, y el propio framework escribe el texto de la respuesta final real del agente en el estado de sesión por ti, ya saltándose correctamente cualquier parte `Thought` (confirmado leyendo la propia implementación de `maybeSaveOutputToState`) — la misma lección que el módulo 25.5 tuvo que aprender a mano, aquí ya resuelta por el propio framework:

```go
llmagent.Config{
    OutputKey: outputKey,
    // ...
}

func (c *responseCache) afterAgentCallback(ctx agent.Context) (*genai.Content, error) {
    val, err := ctx.State().Get(outputKey)
    // ... guarda val bajo la clave de caché de este turno ...
}
```

### Un Segundo Bug Real: No Revises Toda la Historia de la Conversación

La primera versión del propio `beforeModelCallback` de este módulo revisaba cada entrada en `llmRequest.Contents` en busca de una palabra bloqueada — pero `Contents` lleva *toda* la historia de la conversación enviada al modelo, no solo el turno actual. Un test real, en vivo y de varios turnos detectó la consecuencia real de inmediato: un intento con palabra bloqueada al principio de una sesión rechazaba permanentemente *cada turno posterior*, incluyendo preguntas de seguimiento perfectamente normales. Un guardarraíl de entrada debería juzgar la entrada actual, no algo que el usuario ya intentó y de lo cual ya siguió adelante — solucionado revisando solo la última entrada de `Contents`, el propio mensaje del turno actual.

### Puntos Clave ✅
- Seis espacios de callback, todos plurales, cubren antes/después de agente, modelo, y herramienta — coincidencias directas con los seis propios de Python, además de soporte nativo para registrar más de uno por etapa.
- Un retorno no nulo desde un callback "before" significa "usa esto en su lugar"; desde un callback "after", reemplaza lo que ya ocurrió.
- Los Callbacks (con alcance de nodo, parte de la propia lógica de un agente) y los Plugins (con alcance de app, transversales) son herramientas genuinamente distintas — los propios Plugins ya construidos en los módulos 25/25.5 son el punto de comparación real, no uno hipotético.
- Un contexto de callback no soporta `Session()` en absoluto — `OutputKey` es la forma real y respaldada de leer la última respuesta real de un agente dentro de un callback, confirmado en vivo, no asumido del enfoque paralelo de Python.
- Un guardarraíl de entrada debe revisar solo el turno *actual*, no toda la historia de la conversación — confirmado de la forma difícil, viendo una sesión real de varios turnos quedar permanentemente bloqueada.

<hr/>

> **¿Vienes de Python?** 🐍 El propio módulo de Python cubre los mismos seis callbacks, el mismo contrato de "devuelve no nulo para anular", y la misma distinción Callbacks-vs-Plugins. Dos cosas vale la pena señalar directamente: el propio `CallbackContext` de Python *sí* expone `.session.events` para el recorrido en reversa que describe el propio laboratorio de este módulo — el equivalente de Go deliberadamente no lo hace, y `OutputKey` es el sustituto real, no una solución alternativa. Por separado, Go trae un port real y funcional del propio `ReflectAndRetryToolPlugin` de Python (`google.golang.org/adk/v2/plugin/retryandreflect`, cuyo propio comentario de documentación enlaza al código fuente de Python que refleja) pero no tiene un equivalente enviado de `ReflectAndRetryModelPlugin`, ni de `on_agent_error_callback`/`on_run_error_callback` (confirmado ausente leyendo cada método que expone `plugin.Plugin`) — ambos son apartes solo-de-Teoría en el propio material de Python también, así que esto refleja exactamente el alcance del propio laboratorio de Python, no una brecha introducida aquí.
