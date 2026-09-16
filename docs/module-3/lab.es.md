# Reto del Laboratorio 3: Construye y Corre el Agente "Eco" (Go) 🦜

## Objetivo

Tu tarea: construir y correr un agente "Eco" simple usando el SDK de Go ADK.

**El Reto:** a diferencia de un chatbot estándar, este agente debe actuar como un **loro**. Nunca debe responder preguntas ni dar información — solo debe repetir la entrada del usuario exactamente como la recibió.

### Comportamiento Esperado

| Entrada del Usuario | Respuesta del Agente (Correcta) | Respuesta del Agente (Incorrecta) |
| :--- | :--- | :--- |
| "Hello!" | "Hello!" | "Hi there, how can I help you?" |
| "What is the capital of France?" | "What is the capital of France?" | "The capital of France is Paris." |
| "12345" | "12345" | "You entered the numbers 1 through 5." |

Esta tabla exacta es lo que `cmd/echo-agent/echo_test.go` verifica automáticamente.

## Tareas del Laboratorio

1. `cmd/echo-agent/main.go` ya existe en este repo, escrito a mano directamente — así es como empieza todo programa de agente en Go.
2. Lee `cmd/echo-agent/prompts/echo_instruction.md` (cargado al iniciar en el caché compartido `internal/infrastructure/prompts` — mira el `init()` de `main.go` — guardado como un archivo de texto plano en vez de una constante de string en Go, para que sea fácil de leer y editar). Fíjate en lo explícita y repetitiva que es ("never answer," "echo the question itself," "do not add commentary"). Esto no es exagerado: el modelo por defecto tiene capacidad de razonamiento y va a intentar ser "útil" a menos que se le diga claramente, más de una vez, que no lo haga. 😅
3. **Estrategia de instrucción:** si cambias la instrucción, mantenla al menos igual de explícita — una instrucción más suave corre el riesgo de que el modelo responda en vez de hacer eco.
4. No se necesita configuración de `.env` para el camino por defecto (Ollama local). ¿Quieres probar contra Gemini en su lugar? Copia `.env.example` a `.env` y define `MODEL_TYPE=gemini` más `GOOGLE_AI_STUDIO_API_KEY`.
5. Corre el agente:

   > **Elección de puerto:** este laboratorio usa `9091` en vez del puerto por defecto del launcher de ADK, `8080` — `8080` suele estar ocupado por otras herramientas de desarrollo local (confirmado en la propia máquina de esta sesión: el proxy de Docker Desktop ya estaba escuchando en él). Si `9091` también está ocupado en tu máquina, revisa con `lsof -i :9091` (macOS/Linux) y elige cualquier otro puerto libre, ajustando cada URL de abajo **y** la flag `-api_server_address` de abajo para que coincida.
   >
   > **Un problema real y confirmado: `--port` solo no alcanza.** `--port` solo cambia en qué puerto se enlaza el propio servidor — el frontend de la Dev UI aprende dónde llamar a la API desde una flag *separada*, `webui`'s own `-api_server_address`, que por defecto usa el string hardcodeado `http://localhost:8080/api` sin importar `--port`. Si te la saltas, la Dev UI siempre va a intentar llamar al puerto 8080 sin importar en qué puerto realmente iniciaste el servidor — confirmado en vivo: `curl .../ui/assets/config/runtime-config.json` devolvió `{"backendUrl":"http://localhost:8080/api"}` incluso cuando el servidor se inició con `--port 9091`. Por eso el comando de abajo pasa `-api_server_address` explícitamente, justo después de la palabra clave `webui`.

   ```bash
   go run ./cmd/echo-agent web --port 9091 webui -api_server_address http://localhost:9091/api api
   ```
   Nota la posición de las flags: las propias flags de `web` (como `--port`) van directamente después de `web`, y luego cada palabra clave de sub-launcher seguida de sus propias flags (`webui`'s `-api_server_address` aquí). **Tanto `webui` como `api` son obligatorios** — el frontend de la Dev UI llama a la API REST para todo lo que no sea su página estática, así que `webui` solo inicia una UI que no puede realmente interactuar con el agente (confirmado en vivo: `/api/list-apps` da 404 sin `api` también registrado).
6. Abre `http://localhost:9091/ui/` e interactúa con el agente para verificar que pasa la tabla de Comportamiento Esperado de arriba. 🎯

   > **Rareza conocida, no un bug:** la Dev UI muestra la respuesta cruda del modelo, incluyendo su razonamiento paso a paso, antes o junto a la respuesta ecoada — la propia UI del launcher no filtra los rastros de razonamiento como sí lo hace el código de este repo en otras partes. Si ves texto de razonamiento visible, eso es esperado con el modelo por defecto de este curso.

### Alternativa: Modo Consola

```bash
go run ./cmd/echo-agent console
```

Un chat CLI sin navegador — escribe tu entrada directamente, mira la respuesta del agente en línea. La misma rareza del rastro de razonamiento aplica acá también.

### Verificando Automáticamente

```bash
go test ./cmd/echo-agent/... -v
```

Esto corre los tres casos exactos de la tabla de Comportamiento Esperado contra el modelo local real y verifica una coincidencia exacta — evitando por completo el renderizado de la UI del launcher (y su rareza del rastro de razonamiento), de la misma forma en que lo hacen las pruebas de `cmd/verify-setup`.

## Preguntas de Autorreflexión 🤔
- El SDK de ADK en Go no tiene definición de agente basada en YAML — solo la forma programática `llmagent.Config`. ¿Qué ganas o pierdes comparado con las dos opciones de Python?
- ¿Por qué necesita la instrucción del agente eco ser tan explícita y repetitiva, cuando una instrucción más simple "funciona" la mayoría de las veces?
- Tanto la Dev UI como el modo consola muestran texto de razonamiento crudo del modelo. ¿Qué necesitarías cambiar (en el código de este repo, no en el SDK) para filtrar eso de cara al usuario final, de la misma forma en que la prueba automatizada ya lo hace para propósitos de verificación?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `cmd/echo-agent/prompts/echo_instruction.md` y el `init()` de `main.go` (cómo llega al caché), luego las funciones `runEcho`/`firstAnswerText` de `echo_test.go` (cómo la prueba las obtiene y usa) — ese es todo el mecanismo, de principio a fin.
