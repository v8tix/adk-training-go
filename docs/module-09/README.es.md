# Módulo 9: Creando Herramientas de Función Personalizadas (Go) 🛠️

## Teoría

### De lo Prediseñado a lo Personalizado

`google_search` del módulo 8 corre completamente adentro del modelo — genial, pero no tienes ni voz ni voto en cómo funciona. Una **herramienta de función personalizada** invierte eso: es tu propio código, llamado localmente por el framework de ADK, con el resultado devuelto al modelo. Así es como conectas un agente a una base de datos propia, algún algoritmo de lógica de negocio, o — en este laboratorio — aritmética básica. 🧮

### `functiontool.New`: El Mecanismo de Auto-Esquema de Go ✨

```go
addTool, err := functiontool.New(functiontool.Config{
    Name:        "add",
    Description: "Adds two numbers together. Use this tool when the user asks to find the sum of two numbers.",
}, add)
```

```go
type AddArgs struct {
    A int `json:"a" jsonschema:"the first number"`
    B int `json:"b" jsonschema:"the second number"`
}

func add(_ agent.Context, args AddArgs) (CalcResult, error) {
    return CalcResult{Status: "success", Result: float64(args.A + args.B)}, nil
}
```

`functiontool.New[TArgs, TResults](cfg, handler)` infiere el *esquema* de parámetros directo del tipo Go de `TArgs`, vía reflexión (`github.com/google/jsonschema-go`) — bastante ingenioso. `Name` y `Description` sí hay que pasarlos explícitamente en `Config`, porque Go no tiene un docstring en tiempo de ejecución del cual robarlos, pero las descripciones por *parámetro* vienen gratis desde el struct tag `jsonschema:"..."` en cada campo.

### Una Forma de Resultado Uniforme

Las cuatro herramientas de la calculadora comparten un solo tipo de resultado, `CalcResult{Status string; Result float64; Error string}` — todas producen el mismo tipo de respuesta, ¿por qué no? Aquí está el detalle importante: `Func[TArgs, TResults]` devuelve `(TResults, error)`, pero un `error` de Go hace fallar *la llamada a la herramienta misma* a nivel de framework — el LLM ni siquiera llega a razonar sobre eso. Entonces la división por cero necesita `(CalcResult{Status: "error", Error: "division by zero"}, nil)` — un resultado estructurado que el modelo sí puede leer y explicar, no un `error` de Go que corta todo de raíz.

**Un bug real que vale la pena conocer — encontrado en revisión:** el tag `json` de `Result` **no debe** tener `omitempty`. ¿Por qué? `omitempty` en un `float64` trata un `0` genuino exactamente igual que "ausente" — así que `add(0, 0)` o `multiply(7, 0)` le entregarían silenciosamente al modelo una respuesta `{"status":"success"}` sin la clave `result` en absoluto, forzándolo a adivinar el número él mismo. 😬 Confirmado vía `json.Marshal(CalcResult{Status: "success", Result: 0})`: con `omitempty`, la clave `result` simplemente desaparece por completo.

### `agent.Context` Siempre Está Ahí

La firma de cada herramienta de función personalizada es `func(agent.Context, TArgs) (TResults, error)` — `agent.Context` es incondicionalmente el primer argumento, la use o no una herramienta dada. Ninguna de las cuatro herramientas de este laboratorio necesita estado de sesión, pero la capacidad siempre está ahí lista para usarse (el módulo 10 la pone en acción de verdad).

### Las Herramientas de Función Personalizadas Funcionan a Través del Backend Local — Confirmado en Vivo, Sin Fallback a la Nube 🎉

A diferencia del agente de visión del módulo 7 y el agente `google_search` del módulo 8, las herramientas de este módulo no necesitan **ningún requisito de Gemini**. `functiontool.New` produce un `genai.FunctionDeclaration` plano — exactamente la única forma de herramienta que `model/openaimodel/tools.go`'s `ensureFunctionToolOnly` ya acepta (solo rechaza herramientas prediseñadas que *no* son de función). Confirmado en vivo: una herramienta `add` corrida contra el modelo local por defecto de este repo (`qwen3.8:27b`) fue genuinamente invocada y devolvió la suma correcta. `cmd/calculator` no fuerza ningún backend — el default local-primero simplemente funciona, sin condiciones.

### Yendo Más Allá: Mezclando una Herramienta Prediseñada y una Personalizada

Adjuntar tanto `geminitool.GoogleSearch{}` como una herramienta de función personalizada al mismo agente falla por defecto — una restricción real de la propia API de Gemini, no algo raro que hace ADK. SÍ existe una solución real y más específica (`IncludeServerSideToolInvocations`), confirmada en vivo que funciona — mira [troubleshooting.md](./troubleshooting.md) para el error exacto y la solución. No la necesitas para este propio laboratorio (solo herramientas de función), pero guárdatela para cuando quieras ambos tipos de herramienta en un agente.

### Puntos Clave ✅
- `functiontool.New[TArgs, TResults](cfg, handler)` envuelve una función Go como herramienta — el esquema de parámetros viene del tipo de `TArgs`; el nombre y la descripción los das tú explícitamente.
- Una división por cero (o cualquier falla a nivel de herramienta) pertenece al resultado estructurado (`CalcResult{Status: "error", ...}`), no a un `error` de Go — un `error` de Go hace fallar la llamada a la herramienta misma, sin darle nada al LLM con qué trabajar.
- Un campo numérico de resultado nunca debe tener un tag JSON `omitempty` — un `0` genuino es una respuesta válida, no una ausente, y `omitempty` lo ocultaría silenciosamente del modelo.
- `agent.Context` es siempre el primer parámetro de una herramienta de función personalizada.
- Las herramientas de función personalizadas funcionan a través del backend local de Ollama, confirmado en vivo — el primer módulo de herramientas que no necesita ningún fallback a la nube. 🎊
- Mezclar una herramienta prediseñada y una personalizada tiene una solución real y más específica vía `IncludeServerSideToolInvocations` — solo Developer API, confirmado vía código fuente.

<hr/>

> **¿Vienes de Python?** 🐍 `functiontool.New` cumple el mismo rol que pasar una función Python normal a la lista `tools` de un agente, con una diferencia real: Go no tiene docstring en tiempo de ejecución, así que `Name`/`Description` son explícitos en vez de leídos de uno. `CalcResult` es el equivalente directo de la forma de resultado dict de Python (`{"status": "success", "result": ...}`, o un dict de error). El `ToolContext` de Python es un parámetro extra opcional; el `agent.Context` de Go siempre está ahí. Y la documentación de Python dice que mezclar una herramienta prediseñada y una función personalizada es imposible fuera de sistemas multi-agente — este SDK de Go tiene una solución real y más específica, probablemente agregada a la API de Gemini después de que esa documentación se escribió.
