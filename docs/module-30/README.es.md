# Módulo 30: Construyendo un Cliente de Streaming Personalizado (Go) 🎙️🔊

## Teoría

### Del Streaming de Texto al Audio Bidireccional

El `/run_sse` del módulo 29 transmite texto en un solo sentido: el cliente manda un mensaje, el servidor transmite la respuesta en una sola respuesta HTTP. El `/run_live` de este módulo es una forma completamente distinta — un WebSocket genuino, abierto durante toda la conversación, con audio fluyendo de entrada y de salida al mismo tiempo. El cliente puede estar grabando mientras el modelo todavía está hablando su respuesta anterior. Eso es lo que compra "bidireccional", y por eso este módulo necesita un transporte nuevo, no solo un endpoint nuevo.

### `runner.RunLive` Es Real — Pero Solo Gemini, Aplicado a Nivel de Interfaz

`google.golang.org/adk/v2/runner.Runner.RunLive` y su envoltorio REST, `RunLiveHandler` (`server/adkrest/controllers/runtime.go`), están completamente implementados — este módulo no es un hueco a rodear. Pero el propio `Flow.RunLive` de `internal/llminternal/base_flow.go` hace esto, textual:

```go
clientProvider, ok := f.Model.(interface{ Client() *genai.Client })
if !ok {
    return nil, nil, fmt.Errorf("model does not support live connection")
}
```

Solo el propio tipo de modelo de `model/gemini` implementa esa interfaz. `model/openaimodel` — con el que corre cada otro módulo de este curso contra Ollama — no lo hace. Este es el primer módulo de toda la serie donde el valor por defecto local-first no tiene ningún lugar en la mesa, confirmado leyendo el propio código fuente del SDK, no inferido de documentación faltante.

### `VertexAILiveModel`: un Tercer Catálogo de Modelos Distinto

`internal/infrastructure/llm.Config` ya tenía `GeminiModel` (la API pública de Gemini) y `VertexAIModel` (el propio catálogo regular de Vertex, módulo 12). Este módulo agrega `VertexAILiveModel`, porque la Live API viene de un **tercer** catálogo separado — confirmado en vivo: el propio valor por defecto de `VertexAIModel` (`gemini-2.5-flash`) no puede abrir una conexión Live en absoluto, mientras que `gemini-live-2.5-flash-native-audio` sí puede, en el mismo proyecto y credenciales de Vertex exactos. Los mismos dos caminos de autenticación de siempre (clave de API de Express Mode, o Project+Location para ADC) — `newVertexAILiveModel` simplemente reutiliza `vertexAIClientConfig` con un nombre de modelo distinto.

### El Protocolo, Tal Como Está Realmente Implementado (No Como se Asume)

Cada afirmación de abajo fue confirmada contra un servidor real corriendo — ya sea leyendo el propio código de `RunLiveHandler` o mediante un intercambio de WebSocket en vivo durante el propio Build de este módulo.

- **Parámetros de query**: `appName`/`userId`/`sessionId` (camelCase, con snake_case como respaldo).
- **Dos formas de mandar audio**: un frame **binario** crudo de WebSocket, auto-envuelto del lado del servidor como `audio/pcm;rate=16000` sin sobre y sin Base64 — el camino que usa el propio cliente de este módulo — o un mensaje de texto JSON con la forma `{"blob": {"mime_type": "...", "data": "<base64>"}}`. El camino binario es una ventaja de eficiencia genuina y específica de Go: el propio laboratorio de Python no tiene esa opción.
- **Una inconsistencia real y confirmada**: el propio campo de ese envoltorio JSON `blob` es `mime_type` (snake_case) — el único campo en snake_case en una forma de solicitud que por lo demás es toda camelCase (`content`, `blob`, `activityStart`, `activityEnd`, `close`). Confirmado en `server/adkrest/internal/models/runtime.go`.
- **La transcripción viene activada por defecto** — el propio `agent.LiveRunConfig` hardcodeado de `RunLiveHandler` activa tanto `InputAudioTranscription` como `OutputAudioTranscription`. La propia respuesta generada por un modelo de audio nativo es solo audio, pero el servidor igual devuelve texto de transcripción real junto con ella — un bonus genuino que este espejo en Go puede mostrar y que el propio laboratorio de Python (sin salida de texto en absoluto) no tiene.

### El Hallazgo Más Importante: La Detección Automática de Actividad Es Quien Decide el Fin del Turno, No `activityEnd`

`genai.ActivityEnd{}` es un mecanismo real y documentado para marcar manualmente el fin de un turno hablado — y es exactamente lo que usaba el primer borrador del cliente de este módulo, al soltar el botón. Estaba mal, y solo el testing en vivo lo detectó.

`RunLiveHandler` nunca configura `agent.LiveRunConfig.RealtimeInputConfig`, así que la Live API se queda en su propio modo por defecto: **detección automática de actividad**, donde el **servidor** observa el propio stream de audio crudo y decide dónde termina el habla. `activityEnd` solo tiene un comportamiento definido una vez que la detección automática se desactiva explícitamente — una perilla que `RunLiveHandler` no le da a quien lo llama ninguna forma de alcanzar desde afuera del SDK. Confirmado en vivo, dos veces, con una conexión real de Vertex AI Live y el mismo audio de habla real exacto:

| Comportamiento del cliente al soltar | Resultado |
| :--- | :--- |
| Mandar `{"activityEnd": {}}` | **Nada.** Sin respuesta, sin error, en silencio, indefinidamente. |
| Mandar ~1s de silencio real de cola en su lugar | Una respuesta completa y real — entrada transcrita, salida transcrita, audio real transmitido. |

Ambos son código de cliente "correcto" según la propia especificación documentada de la API. Solo uno es confiado por **la propia configuración hardcodeada de este servidor** — y no había forma de saber cuál sin mandar audio real de verdad y ver qué volvía. El cliente de este módulo (`sendSilenceTail`, en `cmd/streaming-agent/static/index.html`) manda silencio de cola, no `activityEnd`.

### Un Requisito de Mismo Origen, Confirmado en Vivo

`RunLiveHandler` arma su propio `websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}` sin ninguna sobreescritura de `CheckOrigin`, así que gorilla/websocket cae en su propio comportamiento por defecto: rechazar el upgrade a menos que el `Origin` de la solicitud coincida con su `Host`. Un navegador real siempre manda `Origin`. Esto significa que **ningún header de CORS, configuración de `-webui_address`, ni nada más puede hacer que un cliente de navegador de origen cruzado funcione contra `/run_live`** — el chequeo pasa antes de que nada de eso siquiera se consulte. La propia división en dos procesos del módulo 29 (un servidor de agente separado y un servidor de archivos estáticos separado) es fundamentalmente incompatible con este endpoint. El propio `cmd/streaming-agent` de este módulo sirve su propio cliente estático desde el mismo servidor `net/http` en el que registra `/run_live` — un proceso, un origen, por construcción — la misma forma que usa el propio ejemplo de referencia del SDK (`google.golang.org/adk/v2/examples/bidi`), por la misma razón.

Un segundo gotcha, independiente, y — una vez que evitás el launcher por esta razón — sin efecto práctico, pero confirmado en el camino: el mount de prefijo de path `/api` por defecto del launcher (`cmd/launcher/web/api/api.go`) envuelve cada solicitud en su propio `redirectRewriter`, que embebe `http.ResponseWriter` como campo de interfaz y por eso **no** implementa `http.Hijacker` aunque el propio escritor subyacente real sí lo haga. El propio `Upgrade` de gorilla/websocket hace una aserción de tipo simple `w.(http.Hijacker)`, que falla bajo ese envoltorio — cada upgrade de `/run_live` a través de un prefijo `/api` montado devuelve 500 antes de que se establezca ninguna conexión. `/run_sse` (módulo 29) nunca se vio afectado, porque SSE solo necesita `http.Flusher`, que el envoltorio sí implementa.

### Puntos Clave ✅
- `runner.RunLive` es real y funciona completamente — pero solo con Gemini, chequeado a nivel de interfaz de Go, sin ningún respaldo para el valor por defecto local-first de Ollama de este curso.
- El camino de envío de audio en frames binarios es una simplificación genuina y específica de Go sobre el enfoque de Python, que solo usa Base64.
- El fin de turno para un stream de audio en tiempo real lo decide la propia detección automática de actividad del servidor observando silencio real — no un marcador `activityEnd` mandado por el cliente, confirmado en vivo probando ambos y viendo que solo uno produce una respuesta.
- `/run_live` requiere un cliente de mismo origen por construcción (sin forma de sobreescribirlo); un prefijo `/api` montado lo rompe completamente por una razón no relacionada (un hueco de `http.Hijacker` en el propio envoltorio de CORS del launcher).

<hr/>

> **¿Vienes de Python?** 🐍 El propio laboratorio de Python manda audio exclusivamente como JSON envuelto en Base64 y señala el fin de turno con `{"audio_stream_end": true}`. El cliente de este espejo en Go manda frames binarios crudos (más simple y eficiente) y no señala nada explícito al soltar — simplemente deja de transmitir audio real y deja que un segundo de silencio real haga el trabajo, porque eso es lo que realmente escucha la propia configuración por defecto de este servidor. El laboratorio de Python no se topa con el problema del mount `/api` del launcher porque su propio servidor de referencia no está construido de la misma forma; el propio servidor de un solo proceso y mismo origen de este módulo (reflejando el propio ejemplo oficial del SDK de Go) lo evita por completo al nunca usar ese mount para `/run_live`.
