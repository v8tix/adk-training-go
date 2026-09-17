# Laboratorio 25.5: Construyendo un Guardarraíl de PII "Fail-Closed" (Go) 🛡️🚫

## Objetivo

Construye un Plugin que bloquee un string con forma de tarjeta de crédito filtrado antes de que llegue jamás al usuario, demostrando el patrón Fail-Closed, y luego prueba que realmente detecta una filtración real de un agente real — no solo un evento de test armado a mano.

## Tareas del Laboratorio

### 1. Lee `internal/agents/piiguardrail/agent.go`

Un agente de demostración diminuto — sin ninguna herramienta — instruido para reproducir un número de tarjeta fijo y obviamente falso ante una frase disparadora específica ("test data"). Existe únicamente para darle al guardarraíl algo real que detectar, el mismo rol que cumple el propio `leak_agent` de Python.

### 2. Lee `internal/agents/piiguardrail/pii_guardrail_plugin.go`

El `onEvent` de `piiGuardrail` revisa `event.IsFinalResponse()`, y luego recorre **cada** parte de la respuesta — saltándose cualquier parte marcada como `Thought` — buscando una coincidencia contra una expresión regular simple de tarjeta de crédito. Ante una coincidencia, sobrescribe el texto de esa parte con un mensaje de seguridad e incrementa un contador protegido por un mutex.

**Lee el comentario sobre la verificación de `Thought` con cuidado** — este loop existe porque la primera versión de este archivo solo revisaba `Parts[0]`, y eso dejaba pasar en silencio una filtración real cuando el modelo subyacente es capaz de pensar (el default). Ver la propia sección "Un Bug Real que el Propio Build de este Módulo Detectó" del README para la historia completa.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/pii-guardrail console
```

Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad en el segundo y tercer turno):

```
🛡️  pii-guardrail using qwen3.8:27b

User -> Give me some test data.
🛑 [SAFETY] Response blocked due to policy violation (invocation "e-c266bc88-...", block #1)
Agent -> I'm sorry, but I can't share that information.

User -> What is the capital of Italy?
Agent -> The capital of Italy is Rome.
```

El primer turno dispara la filtración — y el bloqueo — antes de que el usuario vea jamás el número de tarjeta. El segundo turno, no relacionado, pasa completamente sin verse afectado.

### 4. Lee `internal/agents/piiguardrail/pii_guardrail_plugin_test.go`

Tests unitarios puros, sin LLM. Dos merecen atención especial:

- `TestPIIGuardrail_BlocksRealAnswerAmongThoughtParts` — un evento de dos partes armado a mano (una parte de pensamiento, luego la respuesta filtrada real) prueba que el guardarraíl encuentra y bloquea correctamente la *segunda* parte, no la primera.
- `TestPIIGuardrail_DoesNotBlockMatchWithinAThoughtPart` — el caso espejo: una coincidencia *dentro* de una parte de pensamiento se deja deliberadamente intacta, ya que el trabajo del guardarraíl es proteger la respuesta visible para el usuario, no depurar el razonamiento interno.

### 5. Lee `internal/agents/piiguardrail/agent_test.go`

`TestPIIGuardrail_BlocksLeakedCardNumber_{Ollama,Gemini}` corre un `runner.New` real con el agente real y el plugin real conectado en `PluginConfig.Plugins`, y luego dos turnos separados: uno que dispara la filtración (verificando que el número de tarjeta real nunca aparece en la respuesta visible para el usuario, y que el propio `blockedCount` del plugin refleja una intercepción genuina), y uno con una pregunta ordinaria (verificando que el guardarraíl *no* se disparó) — probando tanto el caso positivo como el negativo de forma estructural, no mirando la salida de consola.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `onEvent` se salta las partes marcadas como `Thought` en vez de escanear cada parte sin distinción? ¿Qué saldría mal (o bien) si no lo hiciera?
- La expresión regular del guardarraíl es un patrón simple — ¿qué se necesitaría para detectar un número de tarjeta de crédito escrito con espacios en vez de guiones, y eso es un cambio al plugin o solo al patrón?
- ¿Por qué `blockedCount` está protegido por un mutex aquí, dado que el propio laboratorio de este módulo nunca corre solicitudes concurrentes de verdad? ¿Qué escenario de despliegue real hace que ese mutex sea, aun así, indispensable?
- Si quisieras que este mismo guardarraíl protegiera a un segundo agente completamente distinto, ¿qué necesitarías cambiar en el propio código de ese agente?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/piiguardrail/pii_guardrail_plugin.go` y `agent.go` para el mecanismo real — una expresión regular, un loop sobre las partes de la respuesta real, y un plugin registrado por separado del propio agente en `cmd/pii-guardrail/main.go`.
