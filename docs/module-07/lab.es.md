# Laboratorio 7: Construyendo un Analizador Visual de Catálogo de Productos (Go) 📸

## Objetivo

Construye un agente con capacidad de visión que analiza una foto de producto y escribe una descripción de marketing, usando el patrón de runner del módulo 6 más una pieza nueva: creación explícita de sesión.

## Tareas del Laboratorio

### 1. Lee `internal/agents/visualcatalog/agent.go` 📖

La misma forma que `internal/agents/supportanalyzer` (módulo 6) — `BuildRootAgent(llmModel)` resuelve su propio prompt internamente — pero más simple: sin `OutputSchema`/`OutputKey`, ya que este agente devuelve texto de marketing plano, no JSON estructurado.

### 2. Lee `cmd/visual-catalog/main.go` 📖

Tres cosas para notar:

- **`cfg.ModelType = llm.ModelTypeGemini` está definido en el código, no dejado al `.env`.** La visión lo necesita — el cliente Go del backend local de Ollama no tiene forma de enviar una imagen (mira el README para el hallazgo confirmado). Definir esto en el código significa que el programa no puede tropezar accidentalmente con ese camino de error solo porque el `.env` de alguien tenga `ollama` como default.
- **Un helper estilo `runOnce`, pero con una sesión explícita primero.** `analyzeProduct` llama a `sessionSvc.Create(...)` antes de `r.Run(...)` — a diferencia del helper basado en `runner.NewInMemory` de todos los módulos anteriores, que nunca necesitó esto porque `NewInMemory` crea sesiones automáticamente.
- **La imagen en sí:** `os.ReadFile(imagePath)` + `genai.NewPartFromBytes(imageBytes, "image/jpeg")`, combinado con una parte de texto vía `genai.NewContentFromParts`.

### 3. Ejecútalo ▶️

```bash
go run ./cmd/visual-catalog
```

(Requiere `GOOGLE_AI_STUDIO_API_KEY` en tu entorno o `.env` — mira `.env.example`.)

Salida real y confirmada de este comando exacto (abreviada — las descripciones completas son más largas):

```
🎨 visual-catalog using gemini-3.5-flash

--- Analyzing Product: HEADPHONES-01 ---
📸 Sending image to Gemini...
✅ Description:
### HEADPHONES-01 — Premium Over-Ear Audiophile Headphones
Elevate your listening experience with the HEADPHONES-01, designed for
those who appreciate both exceptional sound quality and timeless design...

--- Analyzing Product: LAPTOP-02 ---
📸 Sending image to Gemini...
✅ Description:
# LAPTOP-02 — High-Performance Sleek Ultrabook
Elevate your daily productivity and creative workflows with the LAPTOP-02...
```

Dos descripciones distintas y precisas, cada una identificando correctamente el producto real en su foto real — audífonos descritos como audífonos, una laptop descrita como laptop, con detalles visuales específicos (el cable enrollado, el escritorio de madera, el chasis de la laptop) sacados de las imágenes reales, no relleno genérico. ¡Genial! 🎉

### 4. Mira fallar el requisito de sesión, a propósito 💥

Comenta la llamada a `sessionSvc.Create(...)` en `analyzeProduct` y vuelve a ejecutar. Deberías ver el error real y confirmado del que trata este módulo:

```
session not found: "sess_HEADPHONES-01"
```

Restaura la llamada a `Create` antes de seguir — esto está pensado para observarse, no para dejarlo roto. 😄

### 5. Bonus (fuera de la lección del SDK de ADK de este curso): confirma que la brecha está en el SDK, no en el modelo 🔍

```bash
go run ./cmd/visual-catalog-local
```

No necesitas `GOOGLE_AI_STUDIO_API_KEY` — este corre completamente contra el default local compartido del repo. Salida real y confirmada:

```
🎨 visual-catalog-local using qwen3.8:27b directly (no ADK agent/runner)

--- Analyzing Product: HEADPHONES-01 ---
✅ Description:
A pair of black over-ear headphones with brushed silver/metallic ear-cup rims...

--- Analyzing Product: LAPTOP-02 ---
✅ Description:
A silver laptop sits open at the center of a light wooden desk...
```

Esto no es una segunda forma de hacer el laboratorio real — se salta `llmagent`/`runner` por completo, llamando a Ollama directamente (vía [kawa](https://github.com/v8tix/kawa), no el SDK de ADK). Existe solo para probar la afirmación del paso 1 desde el otro lado: el servidor del modelo local puede ver imágenes sin problema; la brecha realmente está en `model/openaimodel`.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `cmd/support-analyzer-runner` (módulo 6) nunca necesitó una llamada explícita de creación de sesión, pero `cmd/visual-catalog` sí?
- La limitación de inferencia local de este módulo es específicamente que el cliente del SDK de Go no puede *enviar* una imagen — el servidor del modelo mismo maneja la misma imagen bien cuando se le llama directamente. ¿Por qué importa esa distinción si estuvieras decidiendo si reportar un bug contra el SDK versus contra Ollama?
- Si quisieras analizar un documento PDF en vez de una imagen, ¿qué cambiarías en `analyzeProduct` — la construcción de la `Part`, el tipo MIME, o ambos?
- `cmd/visual-catalog-local` llega al mismo modelo local que este repo ya usa en todos lados, solo sin pasar por `llmagent`/`runner`. ¿Qué perderías si siempre hicieras esto en vez de lo otro — qué te da la capa de agent/runner de ADK que una llamada HTTP cruda no?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/visualcatalog/agent.go` y la función `analyzeProduct` de `cmd/visual-catalog/main.go` para la lección real — ese es todo el mecanismo basado en ADK, de principio a fin. Para el bonus, `DescribeImageLocally` de `internal/agents/visualcatalog/local_vision.go` y `cmd/visual-catalog-local/main.go`.
