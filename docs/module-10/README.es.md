# Módulo 10: Dándole Memoria a los Agentes con Herramientas con Estado (Go) 🧠

## Teoría

### Por Qué las Herramientas Necesitan Memoria

Cada herramienta que tu agente ha usado hasta ahora ha sido sin estado — la llamas, obtienes una salida, se olvida de que existió. Está bien para una calculadora, pero mucho trabajo real necesita una herramienta que recuerde algo entre turnos: el nombre de un usuario, una preferencia, un total acumulado de más atrás en la conversación. Este módulo trata justamente de darle a una herramienta ese tipo de memoria.

### Cada Herramienta Ya Tiene una Puerta de Entrada: `agent.Context`

Cada herramienta de función personalizada que escribes toma `agent.Context` como su primer parámetro — lo has estado pasando desde el Módulo 9 sin hacer mucho con él. Es la puerta de entrada de tu herramienta a la sesión en curso: quién está hablando, qué ya pasó, y — la estrella de este módulo — un pequeño almacén persistente llamado estado de sesión.

```go
func storeName(ctx agent.Context, args StoreNameArgs) (StoreResult, error) {
    if err := ctx.State().Set("user_name", args.Name); err != nil {
        return StoreResult{}, err
    }
    return StoreResult{Status: "success"}, nil
}
```

`ctx.State()` te da exactamente dos operaciones que importan aquí: `Set(key, value)` para recordar algo, `Get(key)` para leerlo después. Ese es todo el vocabulario — bien simple. 👍

### Leyendo de Vuelta lo Que Guardaste

Un `Get` sobre una clave que nunca se estableció no hace panic ni devuelve silenciosamente un valor cero — devuelve un error específico y nombrado, `session.ErrStateKeyNotExist`, para que tu herramienta pueda distinguir "nada aquí todavía" de "algo realmente se rompió". Eso hace fácil escribir un default sensato:

```go
func recallName(ctx agent.Context, _ RecallNameArgs) (RecallResult, error) {
    val, err := ctx.State().Get("user_name")
    if errors.Is(err, session.ErrStateKeyNotExist) {
        return RecallResult{Name: "Stranger"}, nil
    }
    if err != nil {
        return RecallResult{}, err
    }
    name, _ := val.(string)
    return RecallResult{Name: name}, nil
}
```

### Una Herramienta Que No Recibe Nada

`recall_name` no necesita ningún dato del modelo — solo busca algo. Pero toda herramienta igual necesita un struct de argumentos tipado, así que la forma honesta de decir "sin argumentos" es... uno vacío:

```go
type RecallNameArgs struct{}
```

Confirmado en vivo: el modelo la llama correctamente sin ningún argumento, tal como esperarías. ✅

### La Memoria Realmente Sobrevive Entre Turnos 🔥

Aquí está la parte que vale la pena dudar hasta verla con tus propios ojos: ¿el estado que se guarda en un intercambio realmente sigue existiendo varios mensajes después? Confirmado con una corrida real contra el modelo local: un turno ("Hi, I'm Mario.") llama a `store_name`; un turno completamente separado y posterior en la *misma conversación* ("What is my name?") llama a `recall_name` y recibe "Mario" de vuelta. Esto no es el modelo simplemente recordando el texto anterior — es estado real, persistido, ligado a la sesión, y el test de este módulo lo prueba directamente revisando el resultado de la herramienta misma, no solo la respuesta final del modelo.

### Probando la Memoria Sin Correr un Modelo

No necesitas una llamada real a un LLM para probar `storeName`/`recallName` — solo necesitas algo que se comporte lo suficientemente parecido a `agent.Context` para ejercitar la lógica. El SDK trae exactamente eso: `agent.StrictContextMock`, un pequeño test double que embebes y sobrescribes solo en el único método que realmente necesitas. Cualquier cosa que olvidaste sobrescribir hace panic de forma ruidosa en vez de devolver silenciosamente un valor cero falso que podría esconder un bug real — un modo de falla mucho más amigable que "por qué este test está silenciosamente mal":

```go
type fakeContext struct {
    agent.StrictContextMock
    state *fakeState // a tiny map-backed session.State
}

func (c *fakeContext) State() session.State { return c.state }
```

### Puntos Clave ✅
- `agent.Context.State()` le da a una herramienta un pequeño almacén persistente de clave-valor, con alcance de la sesión actual.
- `Get` devuelve el error nombrado `session.ErrStateKeyNotExist` cuando falta una clave, para que una herramienta pueda construir su propio comportamiento por defecto alrededor de ese caso específico.
- Una herramienta sin argumentos igual necesita un struct tipado — uno vacío, ya que no hay otra forma de decir "sin parámetros".
- El estado realmente persiste entre turnos separados en la misma sesión — confirmado en vivo, no asumido. 🎉
- `agent.StrictContextMock` te deja probar una herramienta con estado sin nunca llamar a un modelo real.

<hr/>

> **¿Vienes de Python?** 🐍 El `agent.Context` de este módulo cumple el mismo rol que el `ToolContext` de Python, con una diferencia estructural: la versión de Python es opcional — agregas `tool_context: ToolContext` como parámetro extra solo cuando una herramienta lo necesita, y el modelo nunca lo ve en el esquema. En Go, cada herramienta de función personalizada ya recibe `agent.Context` como su primer parámetro, la use o no — no hay un paso de activación separado. `ctx.State().Get`/`Set` mapean al `get`/asignación tipo dict de `tool_context.state`; la diferencia real es que el `.get(key, default)` de Python toma un valor de respaldo inline, mientras que Go expresa "no encontrado" como el centinela `session.ErrStateKeyNotExist` que tu propio código chequea explícitamente.
