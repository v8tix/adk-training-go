# Solución de Problemas: Módulo 12 (Go) 🔧

### `400 INVALID_ARGUMENT` mencionando `include_server_side_tool_invocations`

**Causa:** construiste un agente con tanto `geminitool.GoogleSearch{}` como una herramienta personalizada, sin activar esa bandera.

**Solución:** activa `GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true` (el enfoque de agente combinado del Paso 3), o vuelve a dividir en dos agentes (el enfoque de composición secuencial de los Pasos 1-2) — nunca una herramienta prediseñada y una personalizada en el `Tools` de un solo agente sin `ToolConfig`.

### El backend local de Ollama rechaza la solicitud

**Causa:** este laboratorio requiere `MODEL_TYPE=gemini` — `cmd/research-assistant/main.go` lo fuerza en código, ya que `google_search` solo funciona en modelos Gemini 2.0+.

**Solución:** verifica que tu `GOOGLE_AI_STUDIO_API_KEY` realmente esté configurada si ves un error de autenticación en vez del rechazo esperado del backend local.
