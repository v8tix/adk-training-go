# Laboratorio 9: Construyendo un Agente "Calculadora" (Go) 🧮

## Objetivo

Construye un agente que hace aritmética básica usando herramientas de función personalizadas.

## Tareas del Laboratorio

### 1. Lee `internal/agents/calculator/tools.go`

Cuatro funciones handler (`add`, `subtract`, `multiply`, `divide`), cada una `func(_ agent.Context, args XxxArgs) (CalcResult, error)`. Fíjate que `divide` devuelve `CalcResult{Status: "error", Error: "division by zero"}, nil` para un denominador cero — un resultado estructurado que el modelo puede leer y explicar, no un `error` de Go que reventaría la llamada a la herramienta misma.

### 2. Lee `internal/agents/calculator/agent.go`

Cada handler se envuelve vía `functiontool.New(functiontool.Config{Name, Description}, handler)`, y las cuatro se adjuntan vía `Tools: []tool.Tool{addTool, subtractTool, multiplyTool, divideTool}`. A diferencia de `internal/agents/researcher` (módulo 8), acá no hay `MODEL_TYPE` forzado — este agente funciona feliz contra el default local.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/calculator console
```

Sin `.env`, sin API key, nada. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad):

```
🧮 calculator using qwen3.8:27b

User -> What is 42 + 118?
Agent -> 42 + 118 = **160**

User -> What is 10 divided by 0?
Agent -> I'm sorry, but division by zero is not defined in mathematics.
No number multiplied by 0 can give 10, so there is no valid result
for 10 ÷ 0. As the calculation tool confirmed, this operation
results in an error.

User -> What is the capital of France?
Agent -> I appreciate the question, but I'm a calculator assistant —
I can only help with arithmetic operations like addition,
subtraction, multiplication, and division. I'm not equipped
to answer general knowledge questions.
```

Tres comportamientos confirmados funcionando: un cálculo real, una explicación elegante de división por cero (no un crash ni un número inventado), y un rechazo elegante de una pregunta fuera de tema — todo en el modelo local, costo cero. 🎉

### 4. Lee `internal/agents/calculator/tools_test.go`

Tests unitarios table-driven directo contra los cuatro handlers — sin LLM involucrado. Esta es la base rápida y determinística de la pirámide de tests de este módulo; `agent_test.go` (siguiente) cubre que el LLM realmente elija y llame las herramientas correctas.

### 5. Lee `internal/agents/calculator/agent_test.go`

`TestCalculator_Adds_Ollama` y `TestCalculator_Adds_Gemini` — el primer módulo de herramientas (después de los agentes solo-nube de 7 y 8) donde ambas variantes pasan de verdad, confirmando el gran hallazgo de este módulo: las herramientas de función personalizadas no necesitan ningún fallback a la nube.

## Preguntas de Autorreflexión 🤔
- Python se apoya mucho en el docstring de una función para que el LLM la entienda. ¿Cuál es el equivalente en Go, dado que las funciones de Go no tienen docstring en tiempo de ejecución? (Mira `functiontool.Config.Description` y los struct tags `jsonschema` en `tools.go`.)
- ¿Por qué es buena práctica que una función de herramienta devuelva un resultado estructurado `{"status": ...}` en vez de lanzar una excepción (Python) o un `error` de Go para algo que puede fallar, como una división?
- ¿Cómo agregarías una nueva herramienta — digamos, `sqrt` — a este agente? ¿Qué escribirías, y dónde?
- Este módulo encontró que mezclar `google_search` con una herramienta de función personalizada tiene una solución real y más específica en este SDK de Go (`IncludeServerSideToolInvocations`) que la documentación de Python ni siquiera menciona. ¿Por qué podría ser riesgoso apoyarse en un hallazgo así en un agente de producción real, comparado con la solución multi-agente documentada?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/calculator/tools.go` y `agent.go` para el mecanismo real — cuatro funciones de herramienta y cuatro llamadas a `functiontool.New`, y esa es toda la implementación.
