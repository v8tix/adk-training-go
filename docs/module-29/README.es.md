# Módulo 29: Introducción a la Integración de UI (Go) 💬🌐

## Teoría

### Por Qué Importa la Integración de UI

Cada agente hasta ahora ha corrido desde una CLI, un test, o el propio Dev UI del ADK. Un producto real orientado al usuario final normalmente necesita su **propia** interfaz — un widget de chat en un producto, un panel de soporte, una app móvil. Sin importar cómo se vea, habla con tu agente de la misma forma: por HTTP, a la propia API REST que este repositorio ya viene corriendo desde el módulo 3.

### El Panorama de Integración de UI

| Enfoque | Mejor Para | Características Clave |
| :--- | :--- | :--- |
| **Protocolo AG-UI** | Apps web modernas (React/Next.js) | Componentes prearmados, soporte oficial |
| **API Nativa de ADK** | Frameworks personalizados (Vue, Angular) | Control total, sin dependencias |
| **Directo, en proceso** | Apps de datos | Sin overhead HTTP |
| **Plataformas de Mensajería** | Bots de equipo (Slack, Teams) | UX nativa de la plataforma |
| **Basado en Eventos** | Flujos de trabajo asíncronos de alta escala | Desacoplado, escalable (Pub/Sub) |

**AG-UI**, desarrollado a través de una alianza oficial entre los equipos de ADK y CopilotKit, es el propio camino recomendado por Google para apps de producción en React/Next.js — componentes prearmados manejan el streaming y el estado por vos. Es una elección de biblioteca de frontend, no una API de Go de backend para la cual este repositorio expone algo especial — no hay nada que construir acá del lado de Go más allá de la propia API REST a la que el propio adaptador de AG-UI llamaría, la misma exacta que este laboratorio usa directamente.

### La API REST que Ya Tenés

`google.golang.org/adk/v2/cmd/launcher/web/api` — usada por cada invocación `web ... api` desde el módulo 3 — es todo el backend que este módulo necesita. Sin código de servidor nuevo, solo un tipo de cliente genuinamente nuevo: una página HTML/JavaScript escrita a mano, llamando al mismo endpoint de streaming `/run_sse` que el propio Dev UI del ADK usa internamente.

```go
config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
l.Execute(ctx, config, os.Args[1:])
```

Córrelo como `web --port=9093 api -webui_address http://localhost:9094` y tenés un backend REST real y con streaming — sin `webui`, sin Dev UI, solo la API a la que un cliente personalizado le habla.

### Una Diferencia de Nomenclatura Real y Confirmada: `/api`, camelCase, y CORS

Tres cosas concretas que un cliente de Go tiene que hacer bien, en las que un cliente del laboratorio de Python no necesitaría pensar de la misma forma:

1. **La API REST está montada bajo `/api` por defecto** — confirmado en la propia bandera `-path_prefix` de `cmd/launcher/web/api/api.go` (por defecto `"/api"`). Cada endpoint que este cliente llama es `http://localhost:9093/api/...`, no la raíz simple donde sirve el propio `adk api_server` de Python.
2. **Los cuerpos de las solicitudes son camelCase** (`appName`, `userId`, `sessionId`, `newMessage`), confirmado en las propias etiquetas struct de `server/adkrest/internal/models/runtime.go` del SDK — el cliente equivalente de Python envía snake_case (`app_name`, `user_id`, `session_id`, `new_message`) para la misma solicitud exacta.
3. **CORS se controla con `-webui_address`, no `--allow_origins`** — confirmado en el mismo `api.go`. Mismo propósito (permitir que una página del navegador en otro origen/puerto llame a la API), nombre de bandera distinto.

### Un Gotcha Real y Confirmado que Crea el Propio Modelo por Defecto de este Curso

Confirmado en vivo, golpeando directamente el endpoint real `/run_sse`: el propio modelo de Ollama con capacidad de pensamiento por defecto de este repositorio (`qwen3.8:27b`) transmite una parte marcada `"thought": true` — su propio razonamiento crudo — **antes** de la respuesta final real, en el mismo evento. Un cliente que renderice el texto de cada parte tal cual mostraría al usuario el monólogo interno del modelo en vez de (o junto a) su respuesta real:

```json
{"content":{"role":"model","parts":[
  {"text":"The user wants me to say hello in exactly three words...","thought":true},
  {"text":"Hello there friend."}
]}}
```

Esta es exactamente la misma lección de "`Parts[0]` no es la respuesta final" que este curso ha llevado desde su primer agente — acá tiene que aplicarse en JavaScript, en el navegador, ya que ahí es genuinamente donde el filtrado tiene que pasar para un cliente de UI. Saltate cualquier parte con `thought === true` antes de agregar su texto al chat.

### Ventajas de SSE para Chat

Los Server-Sent Events transmiten texto al cliente a medida que el modelo lo genera — sin esperar la respuesta completa, sin el overhead de un handshake bidireccional que un UI de chat no necesita (a diferencia de WebSockets). Un `fetch`, un `ReadableStream`, líneas `data: ...` parseadas a medida que llegan.

### La Persistencia de Sesión Es tu Trabajo

El propio cliente del laboratorio genera un `sessionId` nuevo en cada carga de página — el historial de la conversación se pierde genuinamente al refrescar. Una app real lo persiste en `localStorage` o una cookie para que la misma sesión sobreviva a una recarga; este laboratorio deliberadamente deja ese paso fuera de alcance para mantenerse enfocado en la propia mecánica de streaming.

### Puntos Clave ✅
- La API REST que este repositorio viene corriendo desde el módulo 3 (`cmd/launcher/web/api`) es todo el backend que un UI personalizado necesita — sin código de servidor Go nuevo, solo un cliente nuevo.
- Tres diferencias reales y confirmadas respecto al propio cliente de Python: el prefijo de path `/api`, los cuerpos de solicitud en camelCase, y `-webui_address` en vez de `--allow_origins` para CORS.
- El modelo con capacidad de pensamiento por defecto de este repositorio genuinamente transmite partes `thought: true` a través de `/run_sse` — un cliente de navegador tiene que filtrarlas, verificado en vivo contra el endpoint real, no asumido.
- AG-UI/CopilotKit es el camino oficialmente recomendado para una app real de producción en React/Next.js, abstrayendo exactamente la mecánica de streaming/sesión que este laboratorio construye a mano — no hay componente del lado del SDK de Go para construir para eso, ya que es una elección de biblioteca de frontend.

<hr/>

> **¿Vienes de Python?** 🐍 El propio cliente del laboratorio de Python golpea la raíz simple (`/run_sse`, `/apps/...`) con campos de cuerpo en snake_case y `--allow_origins` para CORS — el cliente de este espejo en Go golpea `/api/...` con campos en camelCase y `-webui_address`, por las mismas razones subyacentes exactas. El laboratorio de Python no necesita filtrar partes `thought` porque su propio módulo no discute un modelo con capacidad de pensamiento por defecto de la forma en que lo hace la propia configuración local-first de este curso en Go — la mecánica de SSE en sí es idéntica en ambos lados.
