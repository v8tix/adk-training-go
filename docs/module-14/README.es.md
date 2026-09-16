# Módulo 14: Integrando Herramientas de Terceros (Go) 📦

## Teoría

### Una Herramienta de Terceros Es Solo... una Función 😌

Cada herramienta personalizada desde el módulo 9 ha sido una función Go envuelta con `functiontool.New`. Nada cambia cuando la lógica de esa función llama a un paquete que otra persona escribió y mantiene, en vez de código que escribiste tú mismo:

```go
import gowiki "github.com/trietmn/go-wiki"

func lookupWikipedia(_ agent.Context, args LookupWikipediaArgs) (LookupWikipediaResult, error) {
    summary, err := gowiki.Summary(args.Query, 5, -1, false, true)
    if err != nil {
        return LookupWikipediaResult{Status: "error", Error: err.Error()}, nil
    }
    return LookupWikipediaResult{Status: "success", Summary: summary}, nil
}
```

```go
wikipediaTool, err := functiontool.New(functiontool.Config{
    Name:        "lookup_wikipedia",
    Description: "Looks up a topic on Wikipedia and returns a short summary.",
}, lookupWikipedia)
```

Esa es genuinamente toda la integración. `github.com/trietmn/go-wiki` es un paquete Go real e independientemente mantenido (no es parte del SDK de ADK ni de este curso) que hace el trabajo real de la API de Wikipedia — la herramienta que lo envuelve se ve exactamente igual que `add` de `calculator` en el módulo 9. Sin drama. 🎉

### Por Qué No Existe una Clase Wrapper Dedicada 🤷

`tool.Tool` es una interfaz simple: `Name`, `Description`, `IsLongRunning`, más `Declaration`/`Run` para una ejecutable. `functiontool.New` construye un `tool.Tool` a partir de cualquier función que calce con `func(agent.Context, TArgs) (TResults, error)` — no le importa, ni necesita saber, si el cuerpo de esa función son cinco líneas que escribiste tú o una sola llamada a la librería de alguien más. No hay nada que inspeccionar o traducir más allá de los propios tipos de argumento y resultado de la función, que el sistema de tipos de Go ya describe con precisión.

Este es un hallazgo real y confirmado, no una suposición: el SDK fijado `google.golang.org/adk/v2` no tiene ningún tipo wrapper para adaptar el propio modelo de objeto-herramienta de una librería de terceros a la forma del ADK — y no hace falta ninguno, porque una función Go de un paquete de terceros ya presenta exactamente la misma forma que `functiontool.New` maneja.

### Un Detalle Real que Confirmamos en Vivo 😅

Las llamadas subyacentes a la API de Wikipedia de `github.com/trietmn/go-wiki` fallan con un error real si no defines un User-Agent distintivo — Wikimedia limita la tasa del valor por defecto genérico del paquete, ya que lo comparten todos los usuarios del paquete. `internal/agents/factfinder/tools.go` lo arregla con una línea en su propio `init()` de paquete — ve [troubleshooting.md](./troubleshooting.md) para el error exacto y la solución.

### `google_search` Sigue Sin Poder Mezclarse Con Esto 🚫

Misma restricción de los módulos 9 y 12, sin cambios: una herramienta de función personalizada, envuelta o escrita a mano, sigue contando como herramienta de función para la regla de la API de Gemini de "no mezclar herramientas integradas y personalizadas." `lookup_wikipedia` puede convivir en la misma lista de `Tools` con cualquier otra herramienta de función, pero no junto a `geminitool.GoogleSearch{}` sin también activar `IncludeServerSideToolInvocations` (módulo 12) — aplican las mismas dos opciones: combinarlas con esa bandera, o usar composición secuencial.

### Mirando Más Allá: MCP 🔭

¿Quieres publicar una herramienta para que la consuman *otros* frameworks también, no solo este código Go? `google.golang.org/adk/v2/tool/mcptoolset` es el camino real, soportado por el SDK, para eso: MCP (Model Context Protocol) es un estándar abierto y agnóstico al framework para exponer herramientas a cualquier cliente compatible con MCP, más cercano en espíritu al modelo de "una herramienta, muchos consumidores" de LangChain que el envoltorio directo de funciones que usa este módulo. Construir uno es una lección posterior de este curso (módulos 27-28) — vale la pena saber que existe, no vale la pena construirlo todavía.

### Puntos Clave ✅
- La función de un paquete Go de terceros se envuelve en una herramienta exactamente igual que una escrita a mano — `functiontool.New` no hace ninguna distinción.
- No existe ninguna clase adaptadora dedicada en el SDK de Go para esto, y no hace falta ninguna — el modelo de herramientas de Go no tiene una forma de objeto separada de la cual traducir.
- Un arreglo a nivel de paquete como un User-Agent personalizado va en el propio `init()` del paquete, confirmado en vivo que se respeta en llamadas hechas después desde una función distinta.
- La restricción de mezcla de herramientas integradas de los módulos 9/12 aplica sin cambios a una herramienta de terceros envuelta.
- `tool/mcptoolset` es el camino real para publicar una herramienta a una audiencia más amplia que un solo código — cubierto en módulos posteriores de este curso.

<hr/>

> **¿Vienes de Python?** 🐍 Python necesita `google.adk.integrations.langchain.LangchainTool` específicamente porque una herramienta de LangChain no es una función simple — es su propio modelo de objeto (una subclase de `BaseTool` con atributos `name`/`description`/`args_schema`) que hay que inspeccionar y traducir a la forma del ADK. Go no tiene ese problema de traducción: la función exportada de un paquete Go de terceros ya tiene exactamente la forma que `functiontool.New` espera, así que la clase wrapper que Python necesita simplemente no tiene nada que hacer acá.
