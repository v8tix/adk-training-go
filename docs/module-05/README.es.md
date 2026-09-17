# Módulo 5: Ejecutando e Interactuando con Agentes (Go) 🖥️

## Teoría

### Tres Modos, Un Launcher Componible 🎛️

Ya vienes usando `console.NewLauncher()` y `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())` desde el módulo 3. Ahora vamos a sumar un tercer modo: la API REST sola, sin Dev UI — simplemente saca `webui.NewLauncher()` y deja `api.NewLauncher()`. Los tres modos vienen del mismo `universal.NewLauncher` componible:

```go
l := universal.NewLauncher(
    console.NewLauncher(),
    web.NewLauncher(webui.NewLauncher(), api.NewLauncher()),
)
l.Execute(ctx, config, os.Args[1:])
```

### Un Bug Real que Encontramos en Este Módulo (y lo Arreglamos Retroactivamente) 🐛

Acá va un detalle confirmado en vivo, no asumido: `webui` y `api` **deben registrarse juntos, sí o sí**. El Dev UI es un frontend estático que llama a la API REST para todo lo que no sea renderizar su propia página. Si registras solo `webui`, la página carga, `/health` responde `200` — pero `curl http://localhost:PORT/api/list-apps` da 404, porque no hay ninguna API REST corriendo de verdad. Esto le pasó a `cmd/echo-agent` desde el módulo 3 y a `cmd/support-analyzer` desde el módulo 4 — ambos arreglados como parte del trabajo de este módulo. **Lección: que `/health` responda `200` NO es prueba de que el Dev UI funcione** — anda a revisar una ruta `/api/...`.

### El Servidor REST API, Confirmado Ruta por Ruta 🗺️

`google.golang.org/adk/v2/server/adkrest` te da una superficie REST real para manejar un agente desde cualquier cliente HTTP — confirmado leyendo el código fuente del router directamente:

| Ruta | Método | Propósito |
|---|---|---|
| `/api/list-apps` | GET | Listar los agentes cargados por su `Name` |
| `/api/apps/{app_name}/users/{user_id}/sessions/{session_id}` | POST / GET / DELETE | Crear / inspeccionar / borrar una sesión |
| `/api/apps/{app_name}/users/{user_id}/sessions` | GET | Listar las sesiones de un usuario |
| `/api/run` | POST | Ejecución sin streaming |
| `/api/run_sse` | POST | Ejecución con streaming (Server-Sent Events) |
| `/api/run_live` | GET | Ejecución en vivo/bidireccional |

(`/api` es el prefijo de ruta por defecto — el propio `--help` de `web` documenta `-path_prefix` si alguna vez necesitas cambiarlo.)

### Dos Cosas que Debes Saber sobre el Formato del Request 📋

1. **Los cuerpos del request son camelCase.** `RunAgentRequest{AppName, UserId, SessionId, NewMessage}` (confirmado en `server/adkrest/internal/models/runtime.go`) se serializa como `appName`/`userId`/`sessionId`/`newMessage`. El decoder es estricto además — si te equivocas en las mayúsculas, te va a rechazar con un `400` real nombrando el campo problemático. La forma anidada del mensaje (`role`, `parts`, `text`) sí se queda en minúsculas.
2. **`app_name` en la URL/body es el `Name` propio del agente.** El Support Analyzer de este repo define `llmagent.Config.Name = "support_analyzer_agent"`, así que cada llamada del ciclo de vida de sesión necesita apuntar exactamente a ese string.

### La Vista de Trace: Backend Confirmado, Frontend No Verificado de Forma Independiente 🔍

El módulo 3 dejó la Trace View como "no confirmada". Buena noticia — podemos mejorar eso un poco: `server/adkrest/internal/routers/debug.go` define rutas reales — `/dev/apps/{app_name}/debug/trace/session/{session_id}` y `/dev/apps/{app_name}/debug/trace/{event_id}` — así que los datos que una vista de Trace necesitaría sí están servidos de verdad por el SDK de Go. ¿El bundle compilado del frontend de `webui` realmente lo renderiza como una pestaña visible? Eso no lo confirmamos de forma independiente (implicaría hacer ingeniería inversa de JS minificado, no leer código fuente) — así que la afirmación honesta acá es "la capacidad del backend es real", no "la funcionalidad del UI es idéntica pixel por pixel a la de Python".

### App & Runner, Una Vez Más 🔄

La misma nota de arquitectura de módulos anteriores, ahora ejercitada de una tercera manera: tanto `console` como el Dev UI de `web` usan `runner.NewInMemory` por debajo (confirmado desde el módulo 2/3); el servidor REST API maneja las sesiones por su cuenta a través de la misma interfaz `session.Service`, respaldada por una implementación en memoria por defecto (el launcher registra en el log `"No session service configured. Using an in-memory one..."` al arrancar) — una configuración estilo producción donde el framework es dueño de las sesiones y el estado, no el código que llame a `Run`.

### Puntos Clave ✅
- Un solo launcher componible (`universal.NewLauncher`) te da una CLI sin interfaz, un Dev UI completo, y un servidor REST independiente — elige los sub-launchers que necesites.
- `webui` y `api` deben registrarse (y nombrarse en la línea de comandos) juntos — un bug real en este repo hasta este módulo, ahora arreglado en todos lados donde apareció.
- El JSON de la API REST es camelCase y las sesiones se identifican por el `Name` propio del agente — bueno saberlo antes de que te rechacen tu primer request. 😅
- Las rutas de backend de la Trace View son reales y están confirmadas en el código fuente; el renderizado del frontend no se verificó de forma independiente.

<hr/>

> **¿Vienes de Python?** 🐍 `console.NewLauncher()`, `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())`, y `web.NewLauncher(api.NewLauncher())` (sin `webui`) son los equivalentes directos de `uv run adk run`, `uv run adk web`, y `uv run adk api_server`, respectivamente. Dos diferencias de sintaxis reales que hay que esperar: los cuerpos del request son camelCase, no snake_case (el `app_name` de Python se rechaza directamente), y `app_name` es el campo `Name` propio del agente, no un nombre de carpeta de proyecto al estilo Python.
