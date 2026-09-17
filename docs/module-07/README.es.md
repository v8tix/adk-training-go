# Módulo 7: Procesamiento Multimodal y de Imágenes (Go) 🖼️

## Teoría

### Construyendo un Mensaje Multimodal 🧩

Una imagen se une a un mensaje de la misma forma que el texto — como un `*genai.Part` más dentro del mismo `*genai.Content`. `genai.Part.InlineData` guarda los bytes crudos y el tipo MIME, con un constructor conveniente para armarlo:

```go
imagePart := genai.NewPartFromBytes(imageBytes, "image/jpeg")
msg := genai.NewContentFromParts([]*genai.Part{
    genai.NewPartFromText("What is in this picture?"),
    imagePart,
}, genai.RoleUser)
```

`genai.NewContentFromParts` construye el mensaje multi-parte — texto e imagen lado a lado, enviados en un solo turno.

### Una Brecha Real y Confirmada en la Inferencia Local — Precisa, No Exagerada 🔬

Este módulo descubrió algo que vale la pena describir con precisión, confirmado con dos pruebas en vivo separadas (no asumido en ninguna dirección):

1. **El servidor del modelo puede ver imágenes sin problema.** Enviar la misma foto real directamente al endpoint compatible con OpenAI de Ollama (`curl .../v1/chat/completions` con una parte de contenido `image_url`) devuelve una descripción precisa — el modelo mismo tiene capacidad de visión real. ✅
2. **El cliente Go de este repo para ese backend no puede enviar una.** El constructor de requests de `google.golang.org/adk/v2/model/openaimodel` (`convertContents`) solo tiene casos para partes de texto, function-call, y function-response — una `Part` de imagen cae en `default: return nil, fmt.Errorf("openai: unsupported content part %T", part)`. Confirmado en vivo: enviar la misma imagen a través del propio código `llmagent`+`runner` de este repo (no curl) falla inmediatamente con exactamente ese error, sin importar qué modelo esté cargado. ❌

**La afirmación precisa y correcta es: el cliente de este SDK para modelos locales todavía no puede enviar imágenes — no que Ollama o el modelo subyacente no puedan recibirlas.** ¡Esa distinción importa! Por eso `cmd/visual-catalog` requiere `MODEL_TYPE=gemini`, hardcodeado en su propio `main()` en vez de dejarlo al default compartido de `.env`, así el programa nunca puede tropezar accidentalmente con este error confirmado.

### La Visión No Necesita Configuración Nueva ✨

El mismo camino de Gemini que todo módulo desde el módulo 2 ya usa — `internal/infrastructure/llm.BuildModel`, autenticado con `GOOGLE_AI_STUDIO_API_KEY` — maneja la visión correctamente de fábrica. Probado directamente con una imagen real, devolvió una descripción precisa sin ninguna configuración nueva: sin backend nuevo, sin variables de entorno nuevas, nada extra que configurar.

### La Lección de Sesión Explícita, Confirmada en Vivo 📝

Todo programa `cmd/` de módulos anteriores usó `runner.NewInMemory`, que define `AutoCreateSession: true` — las sesiones simplemente aparecen automáticamente la primera vez que haces `Run` contra un `sessionID` nuevo. El `cmd/visual-catalog` de este módulo usa `runner.New(Config{...})` directamente en cambio, donde `AutoCreateSession` es `false` por defecto. Confirmado probando ambos estados:

```go
// AutoCreateSession left false, no session created first:
// Run(...) → "session not found: \"s1\""

// After sessionService.Create(ctx, &session.CreateRequest{AppName, UserID, SessionID}):
// Run(...) → succeeds
```

> **Yendo Más Allá:** `cmd/visual-catalog-local` prueba la brecha de inferencia local de arriba desde el otro lado, en vivo — envía las mismas dos fotos de producto directamente al servidor local de Ollama, saltándose `llmagent`/`runner` por completo, y recibe descripciones correctas. Esto **no es** una segunda forma de hacer la lección real de este módulo (esa sigue siendo `cmd/visual-catalog` — el punto es que ese camino *necesita* Gemini por la brecha confirmada del SDK) — es un bonus que demuestra que la brecha está específicamente en `model/openaimodel`, el cliente del SDK de Go de este repo para el backend local, no en Ollama ni en el modelo. Usa [kawa](https://github.com/v8tix/kawa) (una librería de llamadas HTTP tipadas con reintentos incorporados) como su capa HTTP en vez del SDK de ADK, ya que este camino está explícitamente fuera del framework que enseña este curso. Mira `internal/agents/visualcatalog/local_vision.go` y `cmd/visual-catalog-local/main.go`.

### Puntos Clave ✅
- `genai.NewPartFromBytes`/`NewContentFromParts` construyen un mensaje multimodal — imagen y texto como partes hermanas del mismo contenido.
- El servidor del modelo del backend local de Ollama puede ver imágenes (confirmado vía una llamada directa a la API); el cliente Go de este SDK para ese backend actualmente no puede enviarlas (confirmado vía la ubicación exacta del código fuente y el error). Mantén estos dos hechos separados — confundirlos en cualquier dirección sería incorrecto.
- Gemini vía el camino existente de `GOOGLE_AI_STUDIO_API_KEY` maneja la visión correctamente, sin necesitar configuración nueva.
- `runner.New` (a diferencia de `runner.NewInMemory`) requiere una llamada explícita a `session.Service.Create` antes del primer `Run` contra una sesión nueva — confirmado reproduciendo el fallo y el arreglo en vivo.
- Bonus, fuera de la lección de ADK: `cmd/visual-catalog-local` confirma que el propio servidor del modelo local maneja las mismas imágenes correctamente, totalmente local y gratis, llamando a Ollama directamente — la brecha realmente está en el cliente de este SDK, no en el backend con el que habla.

<hr/>

> **¿Vienes de Python?** 🐍 `genai.NewPartFromBytes`/`NewContentFromParts` son los equivalentes directos de Go a `types.Part(inline_data=types.Blob(...))`/`types.Content(role=..., parts=[...])`. El laboratorio de Python pide una configuración completa de Vertex AI (`GOOGLE_GENAI_USE_VERTEXAI=1`, `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`) para la visión — el camino más simple de este repo con `GOOGLE_AI_STUDIO_API_KEY` la maneja igual de bien, confirmado en vivo. Y que `runner.New` necesite una llamada explícita a `session.Service.Create` es la misma lección que el `run_async` de Python necesitando un `create_session` explícito — `run_debug` (módulo 6) te ocultó ese paso en ambos lenguajes.
