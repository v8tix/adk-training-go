# Módulo 27: Introducción a MCP y Herramientas con Estado (Go) 🔌🗂️

## Teoría

### Una Herramienta de Función Olvida Todo Entre Llamadas

Cada herramienta personalizada desde el módulo 9 ha sido una función de Go: se ejecuta, devuelve un valor, y olvida que alguna vez corrió. Eso está bien para `add(a, b)`, pero se queda corto en el momento en que una herramienta necesita conservar estado real a través de una conversación — una conexión de base de datos activa, un archivo abierto, un flujo de reserva de varios pasos.

El **Model Context Protocol (MCP)** es un estándar abierto para exactamente esto: un protocolo cliente-servidor que le permite a un agente hablar con herramientas *externas y con estado* en vez de solo funciones dentro del mismo proceso. En vez de escribir una integración a medida para cada servicio, tu agente se conecta a un servidor MCP ya construido y obtiene sus herramientas gratis — descubiertas en tiempo de ejecución, no declaradas en tu propio código.

### `mcptoolset`: El Cliente MCP Real de ADK

`google.golang.org/adk/v2/tool/mcptoolset` es un cliente MCP de Go real y completo — confirmado directamente en su propio código fuente, no asumido de una página de introducción. `mcptoolset.New(mcptoolset.Config{...})` devuelve un `tool.Toolset` que conectas directamente a `llmagent.Config.Toolsets`:

```go
toolset, err := mcptoolset.New(mcptoolset.Config{
    Transport: &mcp.CommandTransport{
        Command: exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", sandboxDir),
    },
})

llmagent.New(llmagent.Config{
    Name:     "filesystem_agent",
    Model:    llmModel,
    Toolsets: []tool.Toolset{tool.FilterToolset(toolset, tool.AllowedToolsPredicate([]string{"list_directory", "read_file"}))},
})
```

Nota `Toolsets`, no `Tools` — un campo separado del slice de herramientas individuales que cada módulo desde el 9 ha usado. Un `tool.Toolset` no declara sus propias herramientas en tiempo de compilación; las *descubre* de un servidor real cuando se llama a `.Tools()`.

### Tres Transportes Reales, Un Solo Toolset

`mcptoolset` envuelve `github.com/modelcontextprotocol/go-sdk/mcp`, el SDK oficial e independientemente mantenido de MCP para Go. Ese paquete trae tres tipos de transporte reales, confirmados leyendo directamente `cmd.go`, `streamable.go`, y `sse.go`:

| Tipo de Go | A qué se conecta |
|---|---|
| `mcp.CommandTransport` | Un servidor local, lanzado como subproceso, hablando por stdin/stdout |
| `mcp.StreamableClientTransport` | Un servidor remoto, por streaming HTTP bidireccional |
| `mcp.SSEClientTransport` | Un servidor remoto, por Server-Sent Events |

Este laboratorio ejercita los primeros dos: un servidor de sistema de archivos local por stdio, y luego un servidor remoto de GitHub por StreamableHTTP.

### La Conexión Es Perezosa (Lazy)

El propio comentario de documentación de `mcptoolset.New` afirma que la sesión se crea de forma perezosa, en la primera solicitud al LLM — confirmado leyendo `set.go`: `New` solo construye una estructura de transporte; nada se conecta ni lanza un subproceso hasta que realmente se llama a `.Tools()`. Esto tiene una consecuencia genuinamente útil: construir un agente alrededor de un toolset de MCP es un paso de construcción puro, sin efectos secundarios, comprobable sin ningún servidor real corriendo — ver `TestBuildRootAgent_Constructs` en `agent_test.go`, que tiene éxito incluso para un directorio sandbox que aún no existe.

### Seguridad: Aísla (Sandbox) Lo Que Lanzas

`mcp.CommandTransport` corre un subproceso real con los mismos privilegios de tu propio programa — `npx -y @modelcontextprotocol/server-filesystem <path>` puede leer y escribir donde sea que apunte `<path>`, y nada más lo detiene. El aislamiento aquí es enteramente el argumento de directorio que le pasas — no una prisión a nivel de sistema operativo que el SDK provea por ti. Siempre apunta un servidor MCP de stdio al directorio más estrecho que el laboratorio realmente necesite, nunca a un directorio home o a la raíz del repositorio.

### Filtrando la Propia Superficie de Herramientas del Servidor

Un servidor MCP puede exponer más herramientas de las que tu agente debería usar. `tool.FilterToolset(toolset, predicate)` restringe lo que el LLM realmente ve — este laboratorio usa `tool.AllowedToolsPredicate([]string{"list_directory", "read_file"})` para permitir exactamente dos de las herramientas del servidor de sistema de archivos, nada más. (Dos nombres más antiguos hacen lo mismo pero ambos están marcados como obsoletos: `mcptoolset.Config.ToolFilter` en favor de `FilterToolset`, y `tool.StringPredicate` en favor de `AllowedToolsPredicate` — usa el par actual.)

### Confirmado en Vivo: Sin Timeout de Sesión MCP con la Caché de `npx` Limpia

Se limpió la caché de ejecución de `npx` de esta máquina para el paquete del servidor de sistema de archivos y se volvió a correr la demo de consola desde cero: sin timeout de sesión, sin necesidad de reintentar — funcionó en el primer intento. Se anota aquí porque es un hallazgo genuino y probado, no traído de ningún otro lado.

### Puntos Clave ✅
- Las herramientas de función estándar no tienen estado; MCP es la forma real y estándar de darle a un agente acceso a herramientas externas *con estado*.
- `mcptoolset.New` construye un `tool.Toolset` a partir de cualquiera de tres transportes reales (`CommandTransport`, `StreamableClientTransport`, `SSEClientTransport`) — van en `llmagent.Config.Toolsets`, un campo separado de `Tools`.
- La sesión MCP se conecta de forma perezosa — construir un agente alrededor de un toolset nunca marca nada por sí mismo, lo cual hace que la construcción sea completamente comprobable con tests unitarios.
- `tool.FilterToolset` + `tool.AllowedToolsPredicate` restringen cuáles herramientas de un servidor realmente ve el LLM.
- Un servidor MCP lanzado por stdio corre con los propios privilegios de tu programa — el aislamiento es tu responsabilidad, expresado como el path de directorio que le pasas, no algo que el SDK imponga por ti.

<hr/>

> **¿Vienes de Python?** 🐍 El propio módulo de Python cubre los mismos tres tipos de conexión (`StdioConnectionParams`, `SseConnectionParams`, `StreamableHTTPConnectionParams`) y el mismo modelo mental de `McpToolset` — los nombres de Go difieren (`mcp.CommandTransport` en vez de `StdioConnectionParams`, `tool.FilterToolset` en vez de `tool_filter=[...]`) pero la arquitectura y la advertencia de seguridad sobre el aislamiento son idénticas. Una diferencia real: el propio laboratorio de Python advierte que la primera solicitud puede topar con un timeout de sesión MCP mientras `npx` descarga el paquete del servidor por primera vez, aconsejando un simple reintento. Se probó ese escenario exacto en vivo en Go (se limpió la caché de ejecución de npx, se volvió a correr desde cero) y no se reprodujo ningún timeout en esta máquina — vale la pena saberlo si te topas con él, pero no es algo que este espejo en Go pueda prometer que nunca pase en cualquier máquina o red.
