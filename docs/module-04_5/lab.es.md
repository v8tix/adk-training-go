# Reto del Laboratorio 4.5: Analizador de Soporte Listo para Producción (Go) 🛡️

## Objetivo

Mejora el Analizador de Soporte del Módulo 4 **en su lugar** con resiliencia de reintentos, manteniendo su contrato de salida estructurada (`SupportAnalysis`, `OutputSchema`, `OutputKey`) completamente funcional.

## Tareas del Laboratorio

1. **Este módulo no toca ni un archivo en `cmd/support-analyzer`.** La política de reintentos vive en el único lugar que cada módulo ya comparte: `internal/infrastructure/llm`. `cmd/support-analyzer` la hereda automáticamente la próxima vez que corra con `MODEL_TYPE=gemini` — ese es todo el sentido de centralizar la configuración del modelo desde el módulo 2 en adelante. 🎯
2. **Lee `internal/infrastructure/llm/resiliency.go`.** `productionRetryOptions()` devuelve un `*genai.HTTPRetryOptions` con `MaxDelay: 10s, ExpBase: 2.0, Jitter: 0.5`, más una lista explícita de `HTTPStatusCodes` (408/429/500/502/503/504 reintentables, cualquier otro 4xx no).
3. **Lee el `geminiClientConfig` de `internal/infrastructure/llm/factory.go`.** Aquí es donde realmente sucede la centralización de la configuración de producción: una función, llamada por `newGeminiModel`, ella misma llamada por cada módulo a través de `llm.BuildModel`.
4. **El switch local-vs-nube ya existe.** `MODEL_TYPE=ollama` (por defecto) o `MODEL_TYPE=gemini`, leído por `internal/infrastructure/llm.LoadConfig()` desde el módulo 2. Configurar `MODEL_TYPE=gemini` y proveer `GOOGLE_AI_STUDIO_API_KEY` te da el camino resiliente de Gemini que agrega este módulo; dejarlo sin configurar mantiene el valor por defecto local de Ollama, sin afectar por los cambios de este módulo.
5. **Verifica la configuración (a nivel de unidad, sin necesitar llamada real a Gemini):**
   ```bash
   go test ./internal/infrastructure/llm/... -v
   ```
   `TestGeminiClientConfig_UsesProductionRetryOptions` demuestra que `geminiClientConfig` conecta la política de reintentos; `TestRetryableStatusCodes` demuestra que cada código de estado en la tabla de arriba se clasifica correctamente. Ninguna necesita `GOOGLE_AI_STUDIO_API_KEY` ni una llamada de red — esto es una garantía de construcción de configuración, no una demostración de reintento en vivo (el laboratorio de Python hace el mismo punto: *"you won't 'see' the retries unless a network error occurs"*).
6. **Confirma que no hay regresión en el contrato del Módulo 4:**
   ```bash
   go test ./cmd/support-analyzer/... -v
   ```
   Pasa sin cambios — `cmd/support-analyzer` nunca toca directamente la construcción del cliente de Gemini, así que el cambio de este módulo es invisible para él excepto cuando realmente se selecciona `MODEL_TYPE=gemini`. ✅

## Preguntas de Autorreflexión 🤔
- ¿Por qué es importante el "Jitter" en una política de reintentos para una aplicación de producción de alto tráfico? (La misma pregunta que el laboratorio de Python — la respuesta no cambia con el lenguaje.)
- El Nivel 3 de Python centraliza la configuración vía subclases. Go no tiene subclases. ¿Qué usa este repo en su lugar, y dónde apareció ese patrón por primera vez en la historia de este repo?
- ¿Por qué una tabla explícita y basada en datos de códigos de estado (el `retryableStatusCodes` de este módulo) le gana a confiar en un valor por defecto implícito del SDK, incluso cuando ese valor por defecto implícito resulta razonable?
- Este repo nunca necesitó un equivalente de LiteLLM. ¿Qué decisión de un módulo anterior hizo eso realidad, y qué habría tenido que existir en su lugar si no hubiera sido así?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/infrastructure/llm/resiliency.go` y `geminiClientConfig`/`newGeminiModel` de `factory.go`, luego `resiliency_test.go` y `TestGeminiClientConfig_UsesProductionRetryOptions` de `factory_test.go` — ese es todo el mecanismo, de principio a fin. No hay ningún archivo en `cmd/` que leer para este módulo; ese es justamente el punto.
