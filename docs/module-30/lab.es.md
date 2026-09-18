# Laboratorio 30: Construyendo un Cliente de Streaming Personalizado (Go) 🎙️🔊

## Objetivo

Construir un cliente de voz real de tipo push-to-talk contra `/run_live`, respaldado por una conexión real a la Gemini Live API sobre Vertex AI — probando el ida y vuelta completo de audio bidireccional, no solo leyendo sobre él.

### Prerrequisitos

Un backend de Vertex AI configurado — `VERTEX_AI_API_KEY` (Express Mode) o `VERTEX_AI_PROJECT`+`VERTEX_AI_LOCATION` (Application Default Credentials), igual que el módulo 12. No hay ningún camino local para este módulo: `runner.RunLive` rechaza cada modelo que el valor por defecto de Ollama de este repositorio construye, chequeado a nivel de interfaz de Go (ver el propio README de este módulo). La prueba de navegador incluida también necesita Google Chrome o Chromium — ver la [sección Tooling del README principal](../../README.md#-tooling) — y se salta limpiamente si no lo encuentra.

## Tareas del Laboratorio

### 1. Lee `internal/infrastructure/llm/config.go` y `factory.go`

`VertexAILiveModel` (por defecto `gemini-live-2.5-flash-native-audio`, variable de entorno `VERTEX_AI_LIVE_MODEL`) y `ModelTypeVertexAILive` — un tercer catálogo de modelos junto a `GeminiModel`/`VertexAIModel`, reutilizando los mismos dos caminos de autenticación de Vertex que construyó el módulo 12.

### 2. Lee `internal/agents/streamingagent/agent.go`

Un agente deliberadamente trivial, sin herramientas — la lección real de este módulo es el protocolo que el cliente le habla, no lo que el agente en sí puede hacer.

### 3. Lee `cmd/streaming-agent/main.go`

Esta **no** es la forma habitual de dos programas `cmd/` basados en el launcher del módulo 29. El propio comentario de documentación explica por qué: el upgrader de WebSocket de `RunLiveHandler` no tiene ninguna sobreescritura de `CheckOrigin`, así que rechaza directamente cualquier cliente de navegador de origen cruzado — ninguna configuración de CORS puede arreglar eso, ya que el chequeo pasa antes de que CORS siquiera se consulte. Este programa en cambio arma su propio servidor `net/http` mínimo (reflejando el propio ejemplo de referencia del SDK, `google.golang.org/adk/v2/examples/bidi`), sirviendo su cliente estático y `/run_live` desde el mismo origen exacto:

```bash
go run ./cmd/streaming-agent
```

Después abrí `http://localhost:9095` en un navegador.

### 4. Lee `cmd/streaming-agent/main_test.go`

Dos tests reales y en vivo, cada uno probando una parte distinta del protocolo contra una conexión real de Vertex AI Live (`adkrest.NewServer` armado directamente, no a través del launcher — el mismo patrón que estableció el módulo 29):

- `TestRunLive_ReturnsRealAudioForATextTurn` — un simple turno de texto JSON alcanza para conseguir una respuesta hablada (en audio) real; no hace falta audio sintetizado para este, ya que la propia *entrada* de un modelo de audio nativo no está restringida a audio, solo su *salida* lo está.
- `TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd` — el hallazgo central del módulo, probado dos veces en la propia prosa del test y una vez en su afirmación real: transmitir el habla real de `testdata/hello.wav` seguida de silencio real produce una respuesta completa y real (entrada transcrita, salida transcrita, audio real) — el mecanismo que usa realmente el propio cliente de este módulo. Mandar `{"activityEnd": {}}` en su lugar (documentado en el test, no reafirmado como un segundo test — un resultado negativo verdadero no tiene forma limpia de distinguirse de un test colgado) no produjo nada en absoluto, en vivo, durante todo el propio Build de este módulo.

Corrélos (se saltan limpiamente sin credenciales reales de Vertex AI):

```bash
set -a && source .env && set +a  # solo hace falta si tus credenciales viven en .env, no en el shell
go test ./cmd/streaming-agent/... -run TestRunLive -v
```

Salida real y confirmada de este comando exacto:

```
=== RUN   TestRunLive_ReturnsRealAudioForATextTurn
--- PASS: TestRunLive_ReturnsRealAudioForATextTurn (2.03s)
=== RUN   TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd
--- PASS: TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd (2.87s)
```

### 5. Lee `cmd/streaming-agent/static/index.html`

El cliente real de push-to-talk, escrito a mano. Cuatro cosas para notar respecto a la propia versión del laboratorio de Python:

- `WS_BASE` se deriva de `window.location` — mismo origen por construcción, no un puerto hardcodeado que mantener sincronizado (ver Paso 3).
- Sin llamada separada de creación de sesión: el propio controlador de este servidor se arma con `AutoCreateSession: true`, confirmado en vivo contra `runner.Runner.getOrCreateSession` — a diferencia del cliente `/run_sse` del módulo 29, que sí necesita ese paso extra.
- El audio se manda como frames binarios crudos de WebSocket (`sendAudioChunk`), no como JSON envuelto en Base64.
- Al soltar el botón, `sendSilenceTail()` manda ~1s de silencio real — no un marcador `{"activityEnd": {}}`. Ver el propio README de este módulo para entender por qué ese es el que realmente funciona.

`static/js/audio-recorder.js`, `pcm-recorder-processor.js`, `audio-player.js`, y `pcm-player-processor.js` están portados tal cual desde el propio cliente de referencia del SDK (`google.golang.org/adk/v2/examples/bidi`) — puro procesamiento de señal con la Web Audio API (conversión Float32↔Int16, un reproductor de buffer circular), no la propia lección de este módulo.

### 6. Corrélo vos mismo 🎙️

```bash
go run ./cmd/streaming-agent
```

Abrí `http://localhost:9095`, mantené presionado el botón **Hold to Talk**, decí algo, y soltá. Deberías escuchar una respuesta hablada real, y ver tanto tus propias palabras transcritas como la respuesta transcrita del modelo aparecer como burbujas de chat — un bonus genuino que este servidor de Go habilita por defecto y que el propio laboratorio de Python (salida solo de audio, sin texto en absoluto) no tiene.

### 7. Lee `cmd/streaming-agent/browser_test.go`

La prueba permanente y de pila completa para el lado del *cliente*, usando un Chrome headless real (`github.com/chromedp/chromedp`) con las banderas de dispositivo de audio falso de Chrome haciendo de micrófono. Afirma que un WebSocket real se conecta y que frames de audio binarios reales se capturan y mandan genuinamente — no que vuelva una respuesta real. Su propio comentario de documentación explica por qué no: confirmado en vivo, dos veces, la propia pipeline de la Web Audio API de Chrome entrega solo silencio desde un dispositivo de captura falso en modo headless, incluso cuando a ese dispositivo falso se le da un archivo `.wav` de habla real vía `--use-file-for-fake-audio-capture` — la amplitud de cada muestra capturada midió exactamente cero. Eso es una limitación genuina de Chromium/headless, no un bug del propio código de este módulo; la prueba real de ida y vuelta de audio vive en el propio test de Go del Paso 4, que escribe bytes genuinamente no-silenciosos directamente por el cable.

```bash
set -a && source .env && set +a
go test ./cmd/streaming-agent/... -run TestVoiceClient -v
```

Salida real y confirmada de este comando exacto:

```
=== RUN   TestVoiceClient_ConnectsAndSendsRealAudio
--- PASS: TestVoiceClient_ConnectsAndSendsRealAudio (6.09s)
```

### Checkpoint

- [ ] `go test ./cmd/streaming-agent/... -race` pasa (los tres tests en vivo, con credenciales reales de Vertex AI Live configuradas)
- [ ] Confirmado a mano: mantener presionado el botón de hablar, decir algo, y soltar produce una respuesta hablada real y burbujas de transcripción reales en el navegador

## Preguntas de Autorreflexión 🤔
- `RunLiveHandler` no te da forma de desactivar la detección automática de actividad desde afuera del SDK. Si la tuviera, ¿preferirías control manual de `activityStart`/`activityEnd` sobre una heurística de silencio de cola para un producto real de push-to-talk? ¿Qué te costaría cada opción?
- El servidor de este módulo evita `cmd/launcher` por completo, armando su propio servidor `net/http` mínimo en su lugar. ¿Qué perderías exactamente al hacer eso para un módulo que **no** tuviera la restricción de mismo origen de `/run_live` forzando la decisión?
- `TestVoiceClient_ConnectsAndSendsRealAudio` deliberadamente no afirma que vuelva una respuesta hablada real. ¿Es eso un test más débil que uno que sí lo haga? ¿Qué necesitarías cambiar sobre **cómo** se le da el audio al navegador para que esa afirmación vuelva a tener sentido?

<hr/>

### ¿Buscás la solución? 🔍

Pista: leé `internal/agents/streamingagent/agent.go`, `cmd/streaming-agent/main.go`, y `cmd/streaming-agent/static/index.html` para el mecanismo real — un agente trivial, un servidor REST+WebSocket de mismo origen, y un cliente de push-to-talk escrito a mano que le habla por `/run_live`.
