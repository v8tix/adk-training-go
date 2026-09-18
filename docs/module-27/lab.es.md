# Laboratorio 27: Conectando un Agente a una Herramienta de Sistema de Archivos con Estado (Go) 🔌🗂️

## Objetivo

Construye un agente cuyas únicas herramientas provienen de un servidor real y externo del Model Context Protocol — un servidor de sistema de archivos lanzado como un subproceso local — y luego prueba que puede listar y leer archivos reales para los cuales nunca definió herramientas él mismo. Una sección de bonus reemplaza el subproceso local por un servidor MCP remoto, alojado en la red.

### Prerrequisitos

- **Node.js y `npx`**: el servidor de sistema de archivos MCP (`@modelcontextprotocol/server-filesystem`) es un paquete de Node.js de terceros. Instala Node.js (que incluye `npx`) desde [nodejs.org](https://nodejs.org/) si aún no lo tienes. Esto es genuinamente necesario — MCP es agnóstico al lenguaje del cliente, pero el servidor prearmado al que este laboratorio se conecta resulta ser un paquete de Node.

## Tareas del Laboratorio

### 1. Lee `internal/agents/mcpfilesystem/agent.go`

`BuildRootAgent(llmModel, sandboxDir)` conecta un toolset de `mcptoolset.New` (vía `mcp.CommandTransport`, lanzando `npx -y @modelcontextprotocol/server-filesystem <sandboxDir>` como subproceso) a `llmagent.Config.Toolsets`, filtrado hasta `list_directory`/`read_file` con `tool.FilterToolset`. `sandboxDir` es un parámetro explícito, no un path oculto a nivel de paquete — esto es tanto más idiomático en Go como lo que hace que el test en vivo de abajo sea completamente autocontenido.

### 2. Lee `internal/agents/mcpfilesystem/agent_test.go`

`TestBuildRootAgent_Constructs` prueba que la construcción nunca marca nada — la sesión MCP es perezosa, confirmado en la sección de Teoría del README. `TestFilesystemMCP_ListsAndReadsFile_{Ollama,Gemini}` construye su propio sandbox aislado con `t.TempDir()` y un archivo de prueba conocido, dirige un `runner.New` real a través de dos turnos, y verifica que tanto el nombre real del archivo como su contenido real aparecen en las propias respuestas del agente. `npxReachable()` hace que ambas variantes en vivo se salten limpiamente (sin fallar) cuando Node.js no está instalado — la misma disciplina que `llm.OllamaReachable` ya estableció para otro tipo de dependencia externa.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/mcp-filesystem console
```

Sin `.env`, sin API key. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad):

```
📂 mcp-filesystem using qwen3.8:27b
Sandboxed to /path/to/adk-training-go/cmd/mcp-filesystem/test_files

User -> What files are in my directory?
Agent -> Your directory contains one file:

- **hello.txt** (file)

Would you like me to read the contents of `hello.txt`?

User -> Great, can you read the content of hello.txt for me?
Agent -> Here's the content of `hello.txt`:

```
Hello from the MCP world!
```

Is there anything else you'd like me to do?
```

Tanto `list_directory` como `read_file` fueron **descubiertas en tiempo de ejecución** del servidor MCP real — ninguna está definida en ningún lado del propio código de este repositorio.

### Punto de Control

- [ ] La salida real capturada muestra al agente listando `hello.txt` por nombre y leyendo su contenido real de vuelta
- [ ] `go test ./internal/agents/mcpfilesystem/... -race` pasa
- [ ] `go test ./internal/agents/mcpgithub/... -race` pasa (los tests de lógica de encabezados siempre corren; el test de red en vivo se salta limpiamente sin `GITHUB_TOKEN`)
- [ ] `go test ./cmd/mcp-filesystem/... -race` pasa (lógica de resolución y creación del directorio sandbox)

### Bonus: Conectando a un Servidor MCP Remoto 🐙

Hasta ahora el agente habló con un servidor MCP *local* lanzado como subproceso. La mayoría de las integraciones del mundo real en cambio se conectan a un servidor que ya está corriendo en algún otro lugar — `internal/agents/mcpgithub` hace exactamente eso, vía `mcp.StreamableClientTransport`, contra el propio servidor MCP alojado de GitHub.

#### 4. Lee `internal/agents/mcpgithub/agent.go`

`githubAuthTransport` es un `http.RoundTripper` personalizado de ~15 líneas que establece dos encabezados (`Authorization: Bearer <token>`, `X-MCP-Readonly: true`) en cada solicitud saliente — suficiente para un Personal Access Token fijo, sin traer la maquinaria de refresco de tokens de `golang.org/x/oauth2` para un problema (tokens que expiran) que un PAT no tiene.

#### 5. Consigue un GitHub Personal Access Token gratuito

Crea uno en [github.com/settings/personal-access-tokens/new](https://github.com/settings/personal-access-tokens/new) con acceso de solo lectura a repositorios, luego establécelo en tu `.env`:

```
GITHUB_TOKEN=tu_token_aquí
```

#### 6. Córrelo

```bash
go run ./cmd/mcp-github console
```

Intenta preguntar: "What are the open issues on google/adk-go?" — sin subproceso, sin `npx`, sin aislamiento local: la llamada a la herramienta va directamente por HTTPS a los servidores de GitHub. (Esta sesión no tuvo un token real disponible para capturar salida en vivo — `TestGithubAuthTransport_SetsBothHeaders` en `internal/agents/mcpgithub/agent_test.go` prueba directamente la lógica de establecer los encabezados, y `TestGitHubMCP_ListsOpenIssues_Ollama` se salta limpiamente sin `GITHUB_TOKEN` establecido, la misma disciplina que los tests opcionales de Gemini de este repositorio ya siguen.)

Algunas cosas cambian cuando el servidor es remoto en vez de local: no hay un ciclo de vida de subproceso que manejar, las fallas de red (timeouts, límites de tasa, un token expirado) se vuelven posibilidades reales que un servidor stdio local nunca tiene, y la superficie de seguridad cambia de "el subproceso tiene acceso al sistema de archivos" a "no pongas la credencial directamente en el código" — por eso el token viene de `.env`, nunca de un literal en el código.

### 7. Lee `internal/agents/mcpgithub/agent_test.go`

`TestGithubAuthTransport_SetsBothHeaders` y `TestGithubAuthTransport_DefaultsToDefaultTransport` prueban la lógica de encabezados del round-tripper y su respaldo cuando no hay transporte base, ambos sin ninguna llamada de red.

## Preguntas de Autorreflexión 🤔
- `mcptoolset.New` nunca marca nada — ¿qué cambiaría en los tests de este laboratorio si las sesiones MCP se conectaran de forma inmediata, en el momento de la construcción, en vez de perezosa?
- `tool.FilterToolset` restringe cuáles herramientas ve el LLM, pero el propio servidor MCP sigue teniendo todas sus herramientas disponibles para quien más se conecte a él. ¿Cuál es el verdadero límite de seguridad aquí, y cuál no lo es?
- Si quisieras que el agente de sistema de archivos también pudiera *escribir* archivos, no solo listarlos y leerlos, ¿qué necesitaría cambiar — algo en el propio código de este repositorio, o algo completamente distinto?
- ¿Por qué `githubAuthTransport` clona la solicitud antes de modificar sus encabezados, en vez de modificar `req` directamente?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/mcpfilesystem/agent.go` y `internal/agents/mcpgithub/agent.go` para el mecanismo real — dos paquetes de agentes pequeños, cada uno envolviendo `mcptoolset.New` con un transporte distinto.
