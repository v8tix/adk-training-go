# Módulo 4.5: Configuración Profesional de Modelos, Resiliencia y Portabilidad (Go) 🛡️

## Teoría

### Una Función, Completamente Configurada

`llmagent.Config.Model` está tipado como `model.LLM` — una interfaz, nunca un string simple con el nombre del modelo — así que un modelo Gemini siempre pasa por `gemini.NewModel(ctx, name, cfg *genai.ClientConfig)`, con `*genai.ClientConfig` como el único lugar donde poner cada configuración de producción: reintentos, timeouts, credenciales, todo. Centralizar esa configuración a través de cada agente en este repo es simplemente una función constructora plana que devuelve un `model.LLM` completamente configurado — sin necesitar un tipo separado ni una subclase. Este repo ya tiene esa función: `newGeminiModel` de `internal/infrastructure/llm` (vía `geminiClientConfig`), llamada a través de `llm.BuildModel` por cada módulo hasta ahora.

### Confirmado en el Código Fuente: Las Opciones de Reintento Son un Mecanismo Real y Autónomo 🔁

`genai.HTTPRetryOptions` (`MaxDelay`, `ExpBase`, `Jitter`, `Attempts`, `InitialDelay`, `HTTPStatusCodes`) no es solo un valor de configuración que el SDK ignora silenciosamente: el propio `doRequest` de `google.golang.org/genai` llama internamente a `retryHTTPRequest(req, retryOptions, client.Do)`, confirmado leyendo el código fuente del SDK. Configurar `genai.ClientConfig.HTTPOptions.RetryOptions` realmente cambia cómo se comporta el cliente ante fallas transitorias — un mecanismo de reintento real y autónomo, no una sugerencia que el SDK podría o no respetar.

```go
func productionRetryOptions() *genai.HTTPRetryOptions {
    maxDelay := 10.0
    expBase := 2.0
    jitter := 0.5
    return &genai.HTTPRetryOptions{
        MaxDelay:        &maxDelay,
        ExpBase:         &expBase,
        Jitter:          &jitter,
        HTTPStatusCodes: retryableStatusCodes,
    }
}
```

### Siendo Explícito Sobre Qué Errores Vale la Pena Reintentar

En vez de dejar `HTTPStatusCodes` en el valor implícito por defecto del SDK, este módulo lo configura explícitamente, como datos:

| Código(s) de estado | ¿Se reintenta? | Por qué |
|---|---|---|
| 408 Request Timeout | Sí | Timeout del lado del servidor — vale la pena reintentar |
| 429 Too Many Requests | Sí | Rate-limited; el backoff ayuda |
| 500 / 502 / 503 / 504 | Sí | Falla del lado del servidor, no un problema de la solicitud |
| Cualquier otro 4xx (400, 401, 403, 404, ...) | No | La solicitud en sí está mal — reintentar desperdicia el presupuesto sin ninguna chance de tener éxito |

Reintentar un error de cliente nunca ayuda — la solicitud en sí está mal, y reintentar solo desperdicia el presupuesto sin ninguna chance de éxito. Ser explícito sobre ese límite, en vez de confiar en un valor por defecto del SDK que no leíste, vale las pocas líneas extra.

### Agnóstico de Modelo por Diseño: Una Fábrica, Cualquier Backend 🔌

Cambiar de backend de modelo sin tocar código que lo llama ya está resuelto en este repo, desde el módulo 2: el registro de fábrica de `internal/infrastructure/llm.BuildModel` (`MODEL_TYPE=ollama`/`gemini`) es el único lugar donde se toma esa decisión. Cada módulo desde entonces ha llamado a la misma función sin importar el backend — el trabajo de reintentos de este módulo se construye directamente sobre eso, en vez de agregar una segunda abstracción encima de una que ya funciona.

### Más Allá del Laboratorio: Paridad de Reintentos También en el Camino de Ollama

El propio registro de fábrica `MODEL_TYPE` de este repo significa que ambos backends comparten un solo punto de llamada (`llm.BuildModel`) — dejar a uno de ellos sin política de reintentos sería una asimetría real y silenciosa, aunque la política de reintentos de arriba es específica de Gemini por sí sola. `openaimodel.ClientConfig.Options` acepta las propias opciones de solicitud de `openai-go`; `ollamaClientConfig` configura `option.WithMaxRetries(4)` y `option.WithMaxRetryDelay(10 * time.Second)` — 5 intentos totales (coincidiendo con el `Attempts: 5` de `productionRetryOptions`) con el mismo techo de 10 segundos de retraso.

**Esto es paridad en cantidad de intentos y techo de retraso, no un comportamiento idéntico byte por byte — confirmado leyendo el propio bucle de reintentos de `openai-go`, no asumido:** `option` no tiene equivalente de `ExpBase`, `Jitter`, ni una lista configurable de `HTTPStatusCodes`. `openai-go` tiene hardcodeado su propio backoff exponencial base 2 con jitter de hasta 25% del retraso, y su propio conjunto de estados reintentables (408, 409, 429, y todo 5xx — cercano, pero no idéntico, a la tabla explícita de este repo, ya que 409 no está en ella y openai-go trata *todo* 5xx como reintentable en vez de una lista nombrada de seis). Ambos son valores por defecto razonables para producción; simplemente no se pueden hacer numéricamente idénticos porque `genai` expone más controles de reintento que `openai-go`.

### Puntos clave ✅
- `*genai.ClientConfig.HTTPOptions.RetryOptions` es un mecanismo de reintento real y autónomo del SDK, confirmado leyendo el propio código de manejo de solicitudes del cliente — no una sugerencia de paso.
- Centralizar la configuración de producción a través de cada agente es simplemente "una función constructora que todos llaman" (`newGeminiModel`/`llm.BuildModel`) — sin necesitar una jerarquía de tipos separada.
- Una clasificación explícita y basada en datos de código de estado transitorio-vs-permanente le gana a confiar en un valor por defecto implícito del SDK — este repo configura `HTTPStatusCodes` él mismo, con cada elección justificada.
- El registro de fábrica `MODEL_TYPE` de este repo ya entrega agnosticismo de modelo, sin cambios desde el módulo 2 — cambiar de backend nunca requiere tocar código que lo llama.
- Ambos backends ahora tienen una política de reintentos — la API de reintentos más limitada de `openai-go` significa que "paridad" es paridad de cantidad de intentos y techo de retraso, no matemática de backoff idéntica.

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python recorre tres niveles de configuración de modelo (un string simple con el nombre del modelo, un objeto `Gemini` con opciones de reintento, y una subclase `Gemini` que centraliza configuraciones a través de muchos agentes), más `LiteLlm` para agnosticismo de backend. Nada de eso mapea 1 a 1: `llmagent.Config.Model` de Go siempre es una interfaz tipada `model.LLM` (sin atajo de string), Go no tiene subclases (así que "centralizar configuración" es simplemente una función constructora), y el propio registro de fábrica `MODEL_TYPE` de este repo ya cumple el rol de `LiteLlm`. `HTTPRetryOptions` es el equivalente directo del parámetro `retry_options` de Python.
