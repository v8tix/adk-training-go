# Laboratorio 11: Construyendo un Agente "Analista de Mercado Global" (Go) 💱

## Objetivo

Construye un agente que obtiene tasas de cambio de moneda en vivo desde una API REST pública real, usando una herramienta declarada desde un spec en vez de escrita a mano.

## Tareas del Laboratorio

### 1. Lee `internal/infrastructure/openapitool/openapitool.go`

`OperationSpec` describe una operación REST: un ID, un resumen, una URL base, un path, y una lista de parámetros. `NewToolset` convierte uno o más de estos en un `tool.Toolset` funcional. Fíjate que `operationTool` (la implementación real de la herramienta) no está exportado — solo trabajas con el spec.

### 2. Lee `internal/agents/marketanalyst/agent.go`

`frankfurterSpec` describe el endpoint real `/latest` de la API de moneda Frankfurter. `BuildRootAgent` lo convierte en un toolset y lo adjunta vía `Toolsets: []tool.Toolset{toolset}` — no el campo plano `Tools` que usaron los módulos 9-10, ya que un toolset es un valor que produce múltiples herramientas, no una sola.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/market-analyst console
```

Sin `.env`, sin API key — la propia API de Frankfurter tampoco necesita autenticación. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad):

```
💱 market-analyst using qwen3.8:27b

User -> Convert 100 USD to EUR.
Agent -> Based on the latest exchange rate (as of September 14, 2026):
**100 USD = 86.57 EUR**

User -> Convert 500 AUD to XYZ.
Agent -> I'm sorry, but "XYZ" is not a recognized currency code, so I
wasn't able to fetch a rate for that conversion. Could you double-check
the code you meant?
```

Una conversión real usando una tasa de cambio real y actual, y una explicación elegante cuando el código de moneda es inválido — no un crash, no una tasa inventada. 👍

### 4. Lee `internal/infrastructure/openapitool/openapitool_test.go`

Tests puros contra un `httptest.Server` local, no la API real de Frankfurter en vivo — las tasas de cambio reales cambian a diario, así que un test fijado a una tasa exacta sería inestable al día siguiente. Estos prueban la lógica de codificación de parámetros y decodificación de respuestas, más los dos caminos de falla de `Run`: una respuesta real de error de API (resultado estructurado) y una falla de red genuina (error de Go).

### 5. Lee `internal/agents/marketanalyst/agent_test.go`

`TestMarketAnalyst_ConvertsCurrency_Ollama` y `_Gemini` — ambos reales, golpeando la API en vivo. Revisan la `FunctionResponse` real de la herramienta de forma estructural (los códigos de moneda están ahí, la tasa es un número positivo), nunca un valor exacto fijado, por la misma razón.

## Preguntas de Autorreflexión 🤔
- ¿Cuáles son las ventajas de describir una herramienta con datos (un spec) en vez de escribir una función Go para ella? ¿Qué pierdes?
- Si dos `OperationSpec`s diferentes en el mismo `Toolset` tuvieran el mismo `OperationID`, ¿qué crees que pasaría? (Revisa el comportamiento de `toolutils.PackTool` ante un nombre duplicado.)
- Las APIs REST reales publican sus propios specs OpenAPI, a menudo en una URL predecible. Si quisieras que `openapitool` construyera un `Toolset` directamente desde uno de esos (en vez de un `OperationSpec` escrito a mano), ¿qué necesitarías agregar?
- ¿Por qué un `404` de la API real pertenece a un resultado estructurado que el modelo puede leer, mientras que una falla de DNS pertenece a un `error` de Go?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/infrastructure/openapitool/openapitool.go` (`OperationSpec`, `NewToolset`, `operationTool`) y `internal/agents/marketanalyst/agent.go` (`frankfurterSpec`, `BuildRootAgent`) — ese es todo el mecanismo, de punta a punta.
