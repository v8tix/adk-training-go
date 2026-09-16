# Solución de Problemas: Módulo 21 (Go) 🛠️

### El orquestador no puede alcanzar al especialista 📡

**Síntoma:** `go run ./cmd/a2a-orchestrator console` falla, se cuelga, o el modelo reporta que no pudo completar la solicitud de investigación.

**Causa:** `cmd/research-specialist-server` no está corriendo, se inició sin los argumentos correctos (`web --port 8001 a2a -a2a_agent_url http://localhost:8001`), o está corriendo en una dirección distinta a la que espera `cmd/a2a-orchestrator` (`RESEARCH_SPECIALIST_URL`, por defecto `http://localhost:8001`).

**Solución:** confirma primero que el servidor está activo, independientemente del orquestador: `curl -s http://localhost:8001/.well-known/agent-card.json` debería devolver JSON real (un `AgentCard` con `name` y `supportedInterfaces`). Si no lo hace, inicia el servidor en su propia terminal (con los argumentos completos `web --port ... a2a -a2a_agent_url ...`) antes de iniciar el orquestador en otra. Si cambiaste el `--port`/`-a2a_agent_url` del servidor, configura un `RESEARCH_SPECIALIST_URL` que coincida en el orquestador.

### Verificar solo `event.Author` no prueba que una llamada remota tuvo éxito ⚠️

**Síntoma:** un test o un helper de inspección de trazas verifica únicamente `event.Author == "research_specialist"` y reporta éxito incluso cuando el especialista era inalcanzable.

**Causa:** confirmado en vivo en este módulo — todos los caminos de fallo en `remoteagent/v2` (una tarjeta de agente que no se puede resolver, un RPC fallido, un timeout) sintetizan su propio evento de error marcado con el mismo nombre del agente wrapper local. Comparar `event.Author` no distingue entre "el agente remoto realmente respondió" y "la llamada falló y este es el evento de error resultante".

**Solución:** verifica también `event.ErrorMessage == ""` y que `event.Content` traiga texto real y no vacío. Revisa `assertDelegatesToRemoteSpecialist` en `internal/agents/a2aorchestrator/agent_test.go` para ver el patrón corregido, y `TestA2AOrchestrator_UnreachableSpecialist_Gemini` para un test que prueba que la corrección realmente distingue los casos.

### `cmd/research-specialist-server` no compila — módulo no encontrado o errores de `go.mod` 🔨

**Causa:** este módulo introduce el primer import directo de este repo de `github.com/a2aproject/a2a-go/v2` (paquetes `a2a`, `a2asrv`) — antes solo estaba presente como dependencia indirecta traída por el SDK fijado.

**Solución:** ejecuta `go mod tidy` desde la raíz del repo; promoverá la dependencia a directa y actualizará `go.sum` en consecuencia. Es una corrección de una sola vez ya aplicada en el commit de este módulo.

### Un test que usa un servidor HTTP real falla intermitentemente con "port already in use" 🎲

**Causa:** el test levanta su propio listener TCP real para probar el viaje de ida y vuelta A2A genuino (no un mock), y un puerto fijo podría chocar con otro proceso, incluyendo una corrida anterior del test que no limpió bien.

**Solución:** confirmado en el propio `agent_test.go` de este laboratorio: siempre haz el bind con `net.Listen("tcp", "127.0.0.1:0")` (el puerto `0` le pide al sistema operativo cualquier puerto libre) en vez de un número de puerto fijo, y lee la dirección realmente asignada de vuelta desde `lis.Addr()`. Esto es lo que ya hace el test que se entrega — si escribes tu propia variante, mantén este patrón. 👍
