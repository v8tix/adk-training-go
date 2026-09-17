# Laboratorio 5: Explorando Diferentes Modos de Ejecución (Go) 🎮

## Objetivo

Ejecuta e interactúa con el agente **Support Analyzer** (de los módulos 4-4.5) usando los tres modos de ejecución que ofrece el SDK de Go de ADK — Dev UI, CLI sin interfaz, y REST API — incluyendo una buena lección de ciclo de vida de sesión: una ejecución falla sin sesión, funciona después de crear una.

## Tareas del Laboratorio

### 1. `web --port 9091 webui -api_server_address ... api` (Dev UI) 🌐

> **Elección de puerto:** usamos `9091` en vez del puerto por defecto del launcher de ADK, `8080` — `8080` suele estar ocupado por otras herramientas de desarrollo local (mira el laboratorio del módulo 3 para ver una colisión real confirmada en la máquina de esta sesión, cortesía del proxy propio de Docker Desktop). Si `9091` también está ocupado en tu máquina, revisa con `lsof -i :9091` y elige cualquier otro puerto libre — solo ajusta cada URL de este laboratorio **y** el flag `-api_server_address` de abajo para que coincida, incluyendo la Tarea 3, que reutiliza este mismo puerto.
>
> **Un gotcha real y confirmado: `--port` solo no alcanza.** ⚠️ El frontend del Dev UI aprende dónde llamar a la API desde un flag *separado*, el propio `-api_server_address` de `webui`, que por defecto es el hardcodeado `http://localhost:8080/api` sin importar `--port` — confirmado en vivo (mira el README del módulo 3 para la historia completa). Por eso el comando de abajo lo define explícitamente. El modo solo-API de la Tarea 3 no necesita este flag — ahí no hay frontend de Dev UI, `curl` habla directo con la API.

```bash
go run ./cmd/support-analyzer web --port 9091 webui -api_server_address http://localhost:9091/api api
```

**Tanto `webui` como `api` son obligatorios** — el frontend del Dev UI llama a la API REST para todo lo que no sea servir su página estática. Abre `http://localhost:9091/ui/`, chatea con el agente, y prueba la vista de Trace si el frontend la muestra (las rutas de backend son reales — mira el README — aunque no afirmamos haber verificado el UI del frontend de forma independiente).

### 2. `console` (CLI sin Interfaz) ⌨️

```bash
go run ./cmd/support-analyzer console
```

Chatea con el agente directo en tu terminal — sin necesidad de navegador.

### 3. `web --port 9091 api` (Servidor REST API) 🔌

Detén el modo anterior, y luego:

```bash
go run ./cmd/support-analyzer web --port 9091 api
```

Abre una **terminal separada** para actuar como cliente.

**Paso A (El Fallo):** intenta `run_sse` sin crear una sesión primero:

```bash
curl -X POST http://localhost:9091/api/run_sse \
     -H "Content-Type: application/json" \
     -d '{
           "appName": "support_analyzer_agent",
           "userId": "test_user",
           "sessionId": "missing_session",
           "newMessage": {"role": "user", "parts": [{"text": "Hello"}]}
         }'
```

Respuesta real y confirmada: `404`, `failed to find the session: failed to get session: session not found: "missing_session"`.

**Dos cosas para notar sobre el formato del request, confirmadas en vivo:** 👀
- Los campos son **camelCase** (`appName`, `userId`, `sessionId`, `newMessage`) — un nombre de campo en snake_case te va a rechazar con un `400: unknown field` real.
- `appName` es el propio **`Name`** del agente (`support_analyzer_agent`, definido en el `buildRootAgent` de `cmd/support-analyzer/main.go`).

**Paso B (La Solución):** crea la sesión explícitamente:

```bash
curl -X POST http://localhost:9091/api/apps/support_analyzer_agent/users/test_user/sessions/test_session
```

Respuesta real y confirmada: `200`, `{"id":"test_session","appName":"support_analyzer_agent","userId":"test_user","lastUpdateTime":...,"events":[],"state":{}}`.

**Paso C (El Éxito):** envía el mensaje de nuevo, apuntando a la sesión que acabas de crear:

```bash
curl -X POST http://localhost:9091/api/run_sse \
     -H "Content-Type: application/json" \
     -d '{
           "appName": "support_analyzer_agent",
           "userId": "test_user",
           "sessionId": "test_session",
           "newMessage": {"role": "user", "parts": [{"text": "I am so happy with your service!"}]}
         }'
```

Verifica que recibes un evento SSE `data:` cuyo `actions.stateDelta.last_ticket_analysis` contiene el análisis JSON estructurado (`category`, `sentiment`, `summary`) — la respuesta real y confirmada de este repo para este input exacto fue `{"category": "general", "sentiment": "positive", "summary": "The customer reached out to provide feedback regarding their overall service experience."}`. 🎉

## Preguntas de Autorreflexión 🤔
- ¿En qué escenarios sería más útil la Trace View detallada del Dev UI que la interfaz de chat simple del modo `console`?
- Los comandos `curl` de arriba son un ejemplo simple de un cliente programático. ¿Qué aplicaciones del mundo real podrías construir que interactúen con la API de tu agente de esta manera?
- ¿Por qué el servidor REST API y los comandos `curl` necesitan dos ventanas de terminal separadas? ¿Qué representa esa separación en una arquitectura de aplicación del mundo real?
- Los comandos curl de este laboratorio tuvieron que cambiar respecto a los de Python (campos camelCase, nombre de agente en vez de nombre de proyecto). ¿Qué te dice eso sobre probar un "puerto" contra el servidor real corriendo, en vez de asumir una traducción 1:1 de los comandos de ejemplo de otro lenguaje?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee la composición del launcher en `cmd/support-analyzer/main.go` (`web.NewLauncher(webui.NewLauncher(), api.NewLauncher())`) — ese es todo el mecanismo. No hay código de agente nuevo en este módulo; todo se trata de cómo se ejecuta el mismo agente del módulo 4.
