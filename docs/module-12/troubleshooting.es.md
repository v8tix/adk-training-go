# Solución de Problemas: Módulo 12 (Go) 🔧

### `400 INVALID_ARGUMENT` mencionando `include_server_side_tool_invocations`

**Causa:** construiste un agente con tanto `geminitool.GoogleSearch{}` como una herramienta personalizada, sin activar esa bandera.

**Solución:** activa `GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true` (el enfoque de agente combinado del Paso 3), o vuelve a dividir en dos agentes (el enfoque de composición secuencial de los Pasos 1-2) — nunca una herramienta prediseñada y una personalizada en el `Tools` de un solo agente sin `ToolConfig`.

### El backend local de Ollama rechaza la solicitud

**Causa:** este laboratorio requiere `MODEL_TYPE=gemini` — `cmd/research-assistant/main.go` lo fuerza en código, ya que `google_search` solo funciona en modelos Gemini 2.0+.

**Solución:** verifica que tu `GOOGLE_AI_STUDIO_API_KEY` realmente esté configurada si ves un error de autenticación en vez del rechazo esperado del backend local.

### `401 CREDENTIALS_MISSING: API keys are not supported by this API` (bonus de Vertex AI)

**Causa:** `VERTEX_AI_API_KEY` está configurada con una API key común de Google Cloud (el formato clásico `AIzaSy...`, generada desde la página normal de credenciales de Cloud Console) en vez de una key generada a través de la propia inscripción **Express Mode** de Vertex AI Studio. El PredictionService de `aiplatform.googleapis.com` rechaza de plano una key no inscrita en Express — confirmado en vivo, esto es una restricción real de autenticación de Google Cloud, no un bug en el código de este repo.

**Solución:** ya sea (a) generar una key Express Mode real en `console.cloud.google.com/vertex-ai/generative/express` y usar esa en su lugar — no toda cuenta de Google Cloud existente puede acceder a este flujo de inscripción — o (b) borrar `VERTEX_AI_API_KEY` y usar Application Default Credentials en su lugar: configura `VERTEX_AI_PROJECT` + `VERTEX_AI_LOCATION`, luego corre `gcloud auth application-default login`.

### `404 NOT_FOUND: Publisher model ... was not found` (bonus de Vertex AI)

**Causa:** `VertexAIModel` (o una sobreescritura manual) nombra un modelo que no está en el catálogo de modelos publisher de Vertex AI para el proyecto/región dado — confirmado en vivo: `gemini-3.5-flash` (el propio valor por defecto de `GeminiModel` de este repo, usado por la API pública de Gemini) no existe en Vertex, aunque la familia del modelo suene igual. El propio catálogo de modelos de Vertex AI usa identificadores distintos y a menudo más rezagados que los de la API pública de Gemini.

**Solución:** usa el propio valor por defecto de `VertexAIModel` (`gemini-2.5-flash`, confirmado en vivo que funciona), o verifica qué modelos están realmente disponibles para tu proyecto/región antes de sobreescribir `VERTEX_AI_MODEL` a otra cosa.
