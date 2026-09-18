# Módulo 28: Construyendo una Herramienta MCP Personalizada (Go) 🛠️🔌

## Teoría

### De Consumidor a Proveedor

El módulo 27 hizo de tu agente un **cliente** MCP — conectándose a un servidor que alguien más construyó. Este módulo invierte la dirección: tú construyes el **servidor**. Una vez que tu propio servicio habla MCP, es utilizable por cualquier cliente compatible con MCP, no solo por tus propios agentes de ADK — un componente genuino, independiente, direccionable por IA.

### Un Servidor Necesita Dos Cosas: un Menú de Herramientas, y una Forma de Ejecutarlas

`github.com/modelcontextprotocol/go-sdk/mcp` es el mismo paquete del que ya depende el propio `mcptoolset` del módulo 27 — es un SDK real y completo para ambos lados de MCP, cliente y servidor. Construir un servidor significa tres llamadas:

```go
server := mcp.NewServer(&mcp.Implementation{Name: "shopping_cart_mcp_server"}, nil)

mcp.AddTool(server, &mcp.Tool{
    Name:        "add_item_to_cart",
    Description: "Adds an item to the shopping cart.",
}, addItem)

if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
    log.Fatal(err)
}
```

`mcp.NewServer` construye el servidor. `mcp.AddTool` registra un handler de herramienta tipado — esto es el equivalente directo del par `@app.list_tools()` + `@app.call_tool()` de Python, pero como una sola llamada en vez de dos funciones decoradas. `server.Run` con `&mcp.StdioTransport{}` sirve por stdin/stdout — la contraparte del lado servidor de `mcp.CommandTransport`, que el módulo 27 ya usó desde el lado cliente.

### Una Victoria Real de Go: Sin Schema Escrito a Mano

El propio laboratorio de Python escribe a mano el `inputSchema` de cada herramienta como un diccionario JSON-Schema crudo. El propio `AddTool` de Go no necesita nada de eso — confirmado en su propio comentario de documentación: el JSON Schema de entrada (y salida) se *infiere automáticamente* de los propios tipos struct de argumento y retorno de tu handler, usando las etiquetas struct `jsonschema` para las descripciones de propiedades:

```go
type AddItemArgs struct {
    Item string `json:"item" jsonschema:"the item to add to the cart"`
}

func addItem(ctx context.Context, req *mcp.CallToolRequest, args AddItemArgs) (*mcp.CallToolResult, AddItemResult, error) {
    // ...
}
```

Este es exactamente el mismo mecanismo que `functiontool.New` ha usado para las herramientas normales de ADK desde el módulo 9 — un struct tipado, un schema inferido, sin duplicación entre el schema y el código.

### Devolviendo Salida Estructurada, Automáticamente

Un handler puede devolver `nil` para `*mcp.CallToolResult` y un valor `Out` real y tipado en su lugar — el propio comentario de documentación de `CallToolResult` confirma que el framework auto-completa `Content` con el texto JSON de ese valor (y también establece `StructuredContent`). Sin `json.Marshal` manual ni envolver en un `TextContent`, a diferencia del laboratorio de Python.

### El Estado Vive en el Servidor, a Través de Cada Llamada del Cliente

El servidor de este módulo mantiene un carrito de compras — un `[]string` simple, protegido por un mutex desde el principio (la misma disciplina que los trackers de estado compartido de este repositorio han seguido desde el módulo 25; un servidor real puede recibir llamadas concurrentes de más de una sesión). Cada llamada a `add_item_to_cart` lo muta; cada llamada a `view_cart` lo lee de vuelta. Ese es todo el punto de construir una herramienta *con estado* de esta manera: el estado genuinamente vive en el proceso del servidor, independiente de qué cliente (o cuántos clientes) se conecten a él.

### Un Gotcha Real y Específico de Go: Stderr No Es Gratis

`mcp.CommandTransport` conecta el stdin/stdout de un subproceso al propio protocolo MCP — confirmado leyendo su propio código fuente, nunca toca `Stderr`. Dejado sin establecer, `exec.Cmd` lo descarta silenciosamente, lo que significa que cada `log.Printf` que hace tu servidor desaparece en la nada para quien lo lance como subproceso. `internal/agents/mcpcart.BuildRootAgent` establece `serverCmd.Stderr = os.Stderr` explícitamente — sin eso, un estudiante nunca vería los propios logs de consola del servidor que el laboratorio de Python explícitamente te dice que revises.

### Yendo Más Allá: Envolviendo un Agente Entero como una Herramienta MCP (Confirmado Ausente)

El propio módulo de Python cierra con una funcionalidad experimental: `to_mcp_server(agent)`, que envuelve un agente entero — su propio ciclo de modelo y todas sus propias herramientas — como una sola herramienta MCP que cualquier host puede llamar. Buscado directamente: no existe equivalente en ninguna parte del código fuente fijado de `google.golang.org/adk/v2`. El único concepto de "servidor MCP" al que los propios ejemplos del SDK de Go hacen referencia es el producto separado **Agent Registry** de Google Cloud — un cliente de catálogo gobernado, no una forma de envolver un agente como herramienta. Un vacío de alcance confirmado, coincidiendo con el propio encuadre "Experimental, preview" de Python para la misma funcionalidad — solo Teoría acá también, nada para construir todavía.

### Puntos Clave ✅
- `mcp.NewServer` + `mcp.AddTool` + `server.Run(ctx, &mcp.StdioTransport{})` son la forma real y completa de construir un servidor MCP en Go — el lado servidor del mismo paquete que el módulo 27 ya usó como cliente.
- `AddTool` infiere el JSON Schema automáticamente de los propios argumentos tipados de tu handler y sus etiquetas struct — sin schema escrito a mano, a diferencia del laboratorio de Python.
- Un handler puede devolver `(nil, salidaTipada, nil)` y dejar que el framework construya el `Content`/`StructuredContent` de la respuesta automáticamente.
- El estado del servidor (un carrito protegido por mutex acá) vive en el proceso del servidor, compartido a través de cada llamada del cliente — el significado real de "herramienta con estado".
- `mcp.CommandTransport` nunca reenvía el `Stderr` de un subproceso lanzado — reenvíalo tú mismo (`serverCmd.Stderr = os.Stderr`) si querés ver los propios logs del servidor.
- No existe equivalente en Go del propio `to_mcp_server(agent)` experimental de Python en esta versión del SDK — confirmado ausente, solo Teoría.

<hr/>

> **¿Vienes de Python?** 🐍 El par de decoradores `@app.list_tools()`/`@app.call_tool()` de Python mapea a una sola llamada `mcp.AddTool` en Go — las mismas dos responsabilidades (anunciar un schema, ejecutar la lógica), una función en vez de dos. Python escribe a mano el `inputSchema` de cada herramienta; Go lo infiere de tu propio struct de argumentos. Ambos lados marcan `to_mcp_server`/envolver-un-agente-como-herramienta como experimental/preview — este espejo en Go de este curso tampoco construye contra eso, por la misma razón.
