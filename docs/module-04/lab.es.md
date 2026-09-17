# Reto del Laboratorio 4: Agente Analizador de Soporte (Go) 🎫

## Objetivo

Construye un agente **Analizador de Soporte** que lea un ticket de soporte al cliente y devuelva un análisis JSON estructurado — `category`, `sentiment`, y un `summary` de una sola oración — en vez de una respuesta en texto plano, y guarde ese análisis en el estado de sesión.

## Tareas del Laboratorio

1. `cmd/support-analyzer/main.go` ya existe en este repo, escrito a mano directamente — el mismo punto de partida que el agente eco del módulo 3.
2. Lee `cmd/support-analyzer/prompts/support_analyzer_instruction.md` (cargado al iniciar en el caché compartido `internal/infrastructure/prompts` — mira el `init()` de `main.go`). Fíjate que enumera los valores exactos permitidos de `category`/`sentiment` en vez de dejarlos abiertos.
3. **Salida Estructurada:** el `buildRootAgent` de `main.go` configura `OutputSchema: supportAnalysisSchema` — un `*genai.Schema` escrito a mano que coincide con los tres campos del struct `SupportAnalysis`. El esquema y el struct son dos declaraciones separadas; mantenlas sincronizadas tú mismo cada vez que cambies una.
4. **Estado de Sesión:** `OutputKey: "last_ticket_analysis"` le dice al SDK que guarde la respuesta JSON del agente en `event.Actions.StateDelta["last_ticket_analysis"]` en el evento de respuesta final.
5. **Por qué el modelo importa aquí:** el `OLLAMA_MODEL` compartido por defecto de este repo (`qwen3.8:27b`, una cuantización GGUF) fue elegido específicamente porque soporta salida restringida por esquema JSON — otras cuantizaciones de la misma familia de modelos no lo hacen (confirmado: Ollama devuelve `501 "structured output is unavailable"` para ellas). No necesitas cambiar `.env` ni credenciales de nube; el valor por defecto simplemente funciona. ✨
6. **Corre y Verifica:**

   > **Elección de puerto:** este laboratorio usa `9091` en vez del puerto por defecto del launcher de ADK, `8080` — `8080` suele estar ocupado por otras herramientas de desarrollo local (mira el laboratorio del módulo 3 para ver una colisión real confirmada en la propia máquina de esta sesión, del proxy de Docker Desktop). Si `9091` también está ocupado en tu máquina, revisa con `lsof -i :9091` y elige cualquier otro puerto libre, ajustando la URL de abajo **y** la flag `-api_server_address` de abajo para que coincida.
   >
   > **Un problema real y confirmado: `--port` solo no alcanza.** El frontend de la Dev UI aprende dónde llamar a la API desde una flag *separada*, `webui`'s own `-api_server_address`, que por defecto usa `http://localhost:8080/api` sin importar `--port` — confirmado en vivo (mira el README del módulo 3 para el hallazgo completo). Por eso el comando de abajo la configura explícitamente.

   ```bash
   go run ./cmd/support-analyzer web --port 9091 webui -api_server_address http://localhost:9091/api api
   ```
   Tanto `webui` como `api` son obligatorios juntos — el frontend de la Dev UI llama a la API REST para todo lo que no sea servir su página estática (mira el módulo 5 para saber por qué). Abre `http://localhost:9091/ui/` y envía *"My screen is completely broken and I'm very angry about it!"* — verifica que la respuesta sea un objeto JSON válido con `category`, `sentiment`, y `summary`.

   > **Rareza conocida, no un bug (igual que en el módulo 3):** la Dev UI y el modo `console` muestran la respuesta cruda del modelo, incluyendo razonamiento paso a paso, antes de la respuesta JSON — esto es el propio renderizado del SDK, no filtrado como sí lo hace el código y las pruebas de este módulo.
7. **Inspecciona el Estado:** después de algunas interacciones, el JSON guardado en `last_ticket_analysis` es lo que un agente o herramienta downstream consumiría en un sistema multi-agente — este módulo no construye ese consumidor, solo demuestra que el valor llega ahí.

### Alternativa: Modo Consola

```bash
go run ./cmd/support-analyzer console
```

### Verificando Automáticamente

```bash
go test ./cmd/support-analyzer/... -v
```

Maneja el agente real directamente vía `runner` (evitando el renderizado de la UI del launcher), lee `event.Actions.StateDelta["last_ticket_analysis"]` del flujo de eventos, hace `json.Unmarshal` en `SupportAnalysis`, y verifica que los tres campos estén poblados para dos tickets con tonos diferentes (y que el sentimiento realmente difiera entre ellos). ✅

## Preguntas de Autorreflexión 🤔
- ¿Por qué es mejor usar `OutputSchema` en vez de simplemente pedirle al modelo "responde en JSON" solo en el texto de la instrucción?
- El SDK configura la restricción de esquema JSON de la solicitud pero no valida ni parsea la respuesta en sí — ¿qué significa eso para cuánto puedes confiar en el contenido de `event.Actions.StateDelta[OutputKey]` sin tu propia verificación con `json.Unmarshal`?
- El requisito de este módulo (salida estructurada) fue la razón por la que el modelo compartido por defecto de este repo cambió a una cuantización GGUF. ¿Qué sugiere eso sobre verificar una nueva capacidad del SDK contra tu runtime real, en vez de asumir que "el backend del modelo ya funcionaba antes, así que va a seguir funcionando"?
- ¿Cómo podría otro agente en un futuro sistema multi-agente consumir el valor de estado `"last_ticket_analysis"`?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `cmd/support-analyzer/main.go` (el struct `SupportAnalysis`, `supportAnalysisSchema`, y `buildRootAgent`), luego la función `analyzeTicket` de `support_test.go` — ese es todo el mecanismo, de principio a fin.
