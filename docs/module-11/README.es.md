# Módulo 11: Integración Empresarial con una Herramienta Declarativa (Go) 🔌

## Teoría

### Declarar una Herramienta en Vez de Escribirla

Cada herramienta de función personalizada hasta ahora ha sido una función Go que escribes a mano, envuelta con `functiontool.New`. Genial cuando conoces los parámetros en tiempo de compilación. ¿Pero qué pasa si quieres exponer *cualquier* operación REST como herramienta — descrita por datos en vez de código (un nombre, una URL, una lista de parámetros) — y obtener una herramienta funcional sin escribir una función wrapper para cada una? Eso es exactamente lo que construye este módulo. 🏗️

### Un Spec, No una Función

`internal/infrastructure/openapitool.OperationSpec` describe una operación REST de forma declarativa:

```go
var frankfurterSpec = openapitool.OperationSpec{
    OperationID: "get_latest_rates",
    Summary:     "Get latest currency exchange rates.",
    BaseURL:     "https://api.frankfurter.dev/v1",
    Path:        "/latest",
    Parameters: []openapitool.ParamSpec{
        {Name: "amount", Type: "number", Description: "The amount to convert", Required: true},
        {Name: "from", Type: "string", Description: "The 3-letter currency code to convert from", Required: true},
        {Name: "to", Type: "string", Description: "The 3-letter currency code to convert to", Required: true},
    },
}
```

`openapitool.NewToolset("frankfurter", frankfurterSpec)` convierte esos datos directo en una herramienta funcional — sin función Go por endpoint. 🎉

### Construyendo una Herramienta Sin un Struct en Tiempo de Compilación

Los genéricos de `functiontool.New` necesitan un struct Go para `TArgs`, conocido en tiempo de compilación. Una herramienta basada en spec no tiene uno — la lista de parámetros es data, decidida en tiempo de ejecución. Entonces `openapitool` construye una herramienta a mano en su lugar: implementando el mismo pequeño conjunto de métodos que toda herramienta necesita (`Name`, `Description`, `Declaration`, `Run`), con `Declaration` armando su JSON Schema desde la lista `Parameters` del spec, y `Run` haciendo la llamada HTTP real y decodificando la respuesta en un `map[string]any` plano — ya que, igual que los parámetros, la forma de la respuesta tampoco se conoce en tiempo de compilación.

La llamada real pasa por [kawa](https://github.com/v8tix/kawa), una librería de llamadas HTTP tipadas ya usada en el camino bonus del propio módulo 7 de este repo. El tipo de respuesta acá tiene que ser un `map[string]any` nombrado, no un struct — todo el punto de este paquete es que la forma real de una respuesta depende de a qué API apunta quien lo llama, así que no hay un conjunto fijo de campos que declarar de antemano. Resulta que ese es exactamente el caso al que no le aplica la decodificación estricta de kawa (que normalmente rechaza un campo JSON no reconocido en un struct destino): un mapa no tiene campos fijos contra los cuales ser "no reconocido", así que cualquier forma decodifica limpiamente. Qué bien. 👌

### Un Valor, Muchas Herramientas: `tool.Toolset`

`llmagent.Config` tiene un campo `Toolsets []tool.Toolset`, separado del campo plano `Tools` que has usado desde el módulo 9 — construido justo para esta forma: un valor que produce varias herramientas. `openapitool.Toolset` lo implementa, así que un agente adjunta todo el conjunto en una línea:

```go
toolset, _ := openapitool.NewToolset("frankfurter", frankfurterSpec)
llmagent.New(llmagent.Config{
    // ...
    Toolsets: []tool.Toolset{toolset},
})
```

Agregar una segunda operación al mismo toolset es solo agregar un segundo `OperationSpec` a la llamada `NewToolset` — sin necesitar una segunda función wrapper.

### Manejando los Errores de una API Real

Una API REST real puede fallar de varias formas, y cada una necesita una respuesta diferente. Confirmado contra la API real de Frankfurter: un código de moneda inválido devuelve una respuesta HTTP normal — `404` con `{"message":"not found"}` — exactamente el tipo de cosa que el modelo debería leer y explicar, no algo que debería tumbar al agente. Una falla de red genuina es diferente: no hay nada sobre qué razonar para el modelo, así que eso se convierte en un `error` de Go real. El `Run` de `openapitool` hace explícita esa separación — la misma distinción que hizo el manejo de división por cero del módulo 9.

`Run` también chequea si falta un parámetro requerido *antes* de siquiera llamar a la API real — atrapado en revisión, ya que un modelo local ocasionalmente se salta uno, y llamar a la API silenciosamente sin él podría dar una respuesta equivocada en vez de un error obvio (Frankfurter mismo pone EUR por defecto si falta `from`, en vez de rechazar la solicitud 😬). Ese chequeo produce el mismo tipo de resultado estructurado que un error de API, así que el modelo tiene una sola forma consistente que leer sin importar cuál de los dos problemas ocurrió.

### Puntos Clave ✅
- Una herramienta se puede construir a partir de datos (un spec) en vez de una función escrita a mano — genial cuando el conjunto de parámetros no se conoce hasta tiempo de ejecución.
- El contrato de despacho de herramientas del framework es solo un pequeño conjunto de métodos (`Name`, `Description`, `Declaration`, `Run`) — `functiontool.New` es una forma conveniente de satisfacerlo, no la única.
- `llmagent.Config.Toolsets` adjunta un valor que produce muchas herramientas, distinto de `Tools`, que las toma de a una.
- La respuesta de error a nivel HTTP de una API real pertenece a un resultado estructurado que el modelo puede explicar; una falla de red genuina pertenece a un `error` de Go.
- Un tipo mapa nombrado (no un struct) esquiva por completo la decodificación estricta de campos no reconocidos de kawa — útil siempre que la forma real de una respuesta no se conozca hasta tiempo de ejecución.

<hr/>

> **¿Vienes de Python?** 🐍 `openapitool.NewToolset` cumple el mismo rol que el `OpenAPIToolset` de Python: entra un spec, salen herramientas funcionando, sin función wrapper por endpoint. No hay un equivalente empaquetado en este SDK de Go, eso sí — confirmado con una búsqueda exhaustiva de `google.golang.org/adk/v2` y sus dependencias, esto es un vacío real y estructural, no solo una diferencia de nombre. `openapitool` lo cierra usando las mismas primitivas `tool.Tool`/`tool.Toolset` que el SDK ya expone, trabajando desde un pequeño struct de Go en vez de un documento OpenAPI JSON/YAML parseado — el mismo alcance que usa el propio laboratorio de Python también, ya que también construye su spec como un dict literal en vez de cargar un archivo de spec real.
