# Laboratorio 24: Un Test de Trayectoria de Camino Dorado para el Agente Calculadora (Go) 🧪📊

## Objetivo

Construye un test de Go que grabe un "camino dorado" — una pregunta, la secuencia exacta y ordenada de llamadas a herramientas que debería disparar, y qué debe contener la respuesta final — y luego prueba que ese test realmente puede detectar una trayectoria equivocada, no solo pasar por coincidencia.

## Antes de Empezar

Este laboratorio **no** modifica nada en `internal/agents/calculator` del Módulo 9 — `agent.go`, `tools.go`, `agent_test.go`, y `tools_test.go` se quedan exactamente como estaban. Estás agregando un archivo nuevo y completamente separado que reutiliza el `BuildRootAgent` público del agente Calculadora ya existente, la misma disciplina de "nunca toques los archivos de la lección real" que este repositorio ha usado para toda demostración adicional anterior.

## Tareas del Laboratorio

### 1. Lee primero `internal/agents/calculator/agent_test.go`

Específicamente `askCalculator` y `assertCalculatorAdds` — esto ya prueba que ocurrió una llamada a una herramienta con el resultado correcto, revisando el propio `FunctionResponse` de la herramienta, no solo el texto final. El archivo nuevo de este laboratorio hace lo mismo, solo que para una *secuencia* de llamadas en vez de una sola.

### 2. Lee `internal/agents/calculator/golden_path_eval_test.go`

- `expectedToolCall`/`goldenPathCase` — un struct simple de Go que hace las veces del `EvalCase` de Python: una pregunta, una lista ordenada de llamadas a herramientas esperadas, y un substring que la respuesta final debe contener.
- `runGoldenPath` — corre `BuildRootAgent` a través de `runner.NewInMemory` + `Run()`, recolectando **cada** evento `FunctionCall` en el orden en que llega (no solo una herramienta nombrada, a diferencia de `askCalculator`).
- `goldenPathCases` — dos casos: una verificación de cordura de una sola herramienta (`"What is 42 + 118?"` → solo `add`), y el caso de trayectoria real: `"First add 10 and 5. Then multiply that result by 2."` → `add(a=10,b=5)` y luego `multiply(a=15,b=2)`, en ese orden exacto. Los propios argumentos de `multiply` genuinamente dependen del propio resultado de `add` (15), así que esto no son dos llamadas que solo parecen secuenciales por casualidad — un orden equivocado o un argumento equivocado aquí es un error real, no una coincidencia.

### 3. Córrelo — sin `.env`, sin API key 🖥️

```bash
go test ./internal/agents/calculator/... -run TestGoldenPath -v
```

Salida real y confirmada de este comando exacto:

```
=== RUN   TestGoldenPath_Calculator_Ollama
=== RUN   TestGoldenPath_Calculator_Ollama/single_tool_call
=== RUN   TestGoldenPath_Calculator_Ollama/multi-step_trajectory:_add_then_multiply,_in_order
--- PASS: TestGoldenPath_Calculator_Ollama (20.23s)
    --- PASS: TestGoldenPath_Calculator_Ollama/single_tool_call (5.29s)
    --- PASS: TestGoldenPath_Calculator_Ollama/multi-step_trajectory:_add_then_multiply,_in_order (14.94s)
=== RUN   TestGoldenPath_Calculator_Gemini
    golden_path_eval_test.go:146: skipping: GOOGLE_AI_STUDIO_API_KEY is not set
--- SKIP: TestGoldenPath_Calculator_Gemini (0.00s)
PASS
```

### 4. Prueba que la aserción es real, no decorativa

No confíes solo en que el test *fallaría* ante un error real — obsérvalo pasar. Intercambia temporalmente las dos entradas del `wantTrajectory` del caso de múltiples pasos (`multiply` primero, `add` segundo) y vuelve a correr:

```bash
go test ./internal/agents/calculator/... -run TestGoldenPath_Calculator_Ollama -v
```

Salida de falla real y confirmada de ese cambio exacto:

```
    golden_path_eval_test.go:134: trajectory[0].tool = "add", want "multiply" — the tool call sequence is out of order or wrong
    golden_path_eval_test.go:134: trajectory[0] (add) argument "a" = 10, want 15
    golden_path_eval_test.go:134: trajectory[0] (add) argument "b" = 5, want 2
    golden_path_eval_test.go:134: trajectory[1].tool = "multiply", want "add" — the tool call sequence is out of order or wrong
    golden_path_eval_test.go:134: trajectory[1] (multiply) argument "a" = 15, want 10
    golden_path_eval_test.go:134: trajectory[1] (multiply) argument "b" = 2, want 5
--- FAIL: TestGoldenPath_Calculator_Ollama (16.53s)
```

Esta es la misma disciplina que el propio Paso 5 de Python ("Test a Failure") recorre rompiendo temporalmente la matemática de la herramienta `add` — aquí, la "rotura" está en la propia expectativa del test, probando que la lógica de comparación en sí es sólida. **Revierte el intercambio antes de continuar.**

### 5. Compara esto con lo que hace el laboratorio de Python en su lugar

Python: abre la Dev UI, ten una conversación, haz clic en "Add current session to eval set", haz clic en "Run Evaluation", lee una tarjeta de Pass/Fail con un `tool_trajectory_score`. Go: escribe un struct literal y un loop. La misma pregunta subyacente ("¿hizo el agente las cosas correctas, en el orden correcto, y dijo algo razonable?"), respondida con las herramientas que cada lenguaje realmente tiene.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `runGoldenPath` recolecta **cada** `FunctionCall`, mientras que el propio `askCalculator` de `agent_test.go` solo rastrea el `FunctionResponse` de una herramienta nombrada? ¿Qué se perdería `askCalculator` que esta versión del laboratorio sí detecta?
- Los argumentos de `multiply` en el caso de múltiples pasos (`15`, `2`) dependen del propio resultado de `add`. ¿Por qué importa esa dependencia para probar que el test no es una coincidencia?
- `assertGoldenPath` usa `strings.Contains` para la respuesta final, no igualdad exacta. ¿Qué se rompería si usara `==` en su lugar?
- Si Go alguna vez lanza un paquete de evaluación real, ¿qué esperarías que cambiara en este archivo de test, y qué probablemente se quedaría igual?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/calculator/golden_path_eval_test.go` directamente — todo el mecanismo está en un solo archivo, construido sobre `BuildRootAgent` y `runner.NewInMemory`, ambos ya familiares desde el Módulo 9.
