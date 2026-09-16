# Solución de Problemas: Módulo 9 (Go) 🔧

### `400 INVALID_ARGUMENT` al mezclar `geminitool.GoogleSearch{}` con una herramienta de función personalizada

**Síntoma:**

```
400 ... Please enable tool_config.include_server_side_tool_invocations
to use Built-in tools with Function calling.
```

**Causa:** adjuntar tanto una herramienta prediseñada (como `google_search`) como una herramienta de función personalizada a la lista `Tools` del mismo agente falla por defecto — una restricción real directo de la propia API de Gemini, confirmada vía código fuente de `genai`.

**Solución:** activa `llmagent.Config.GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true`. Confirmado en vivo: el mismo agente con herramientas mezcladas entonces funciona correctamente, calculando una suma real con una herramienta prediseñada y una personalizada juntas. ✅ Este campo es **solo Developer API** (el propio comentario de documentación de `genai` lo dice directo: "This field is not supported in Vertex AI") — el mismo camino más simple de `GOOGLE_AI_STUDIO_API_KEY` que este repo ya usa, así que no necesitas ninguna credencial extra.
