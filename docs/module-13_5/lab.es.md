# Laboratorio 13.5: Extendiendo ADK con Persistencia Redis Personalizada (Go) 🗄️

## Objetivo

Construye un `session.Service` personalizado respaldado por Redis, inyéctalo en un `runner.Runner` real, y demuestra que sobrevive un reinicio de proceso — ese es el punto real de esta lección.

### Prerrequisitos

Un Redis accesible (por defecto `localhost:6379`, sobreescribible con `REDIS_ADDR`) y Docker (para las pruebas de integración, que usan Testcontainers). No hace falta cuenta cloud — qué alivio. 👍

La forma más rápida de conseguir un Redis accesible: `./scripts/up.sh` desde la raíz del repositorio, que levanta el propio `docker-compose.yml` de este repositorio en segundo plano. `./scripts/stop.sh` lo pausa, `./scripts/clean.sh` lo elimina por completo (junto con sus datos), `./scripts/logs.sh` sigue su salida.

### Paso 1: Los Tipos Concretos

`internal/infrastructure/redissession/session.go` define `redisSession`, `redisState`, y `redisEvents` — vistas planas en memoria que satisfacen `session.Session`, `session.State`, y `session.Events`. No hay ningún tipo concreto exportado en el SDK para embeber acá — cada `session.Service` personalizado construye el suyo desde cero.

### Paso 2: La Implementación del Servicio

Lee `Create`, `Get`, `List`, `Delete`, y `AppendEvent` de `internal/infrastructure/redissession/service.go`. Presta especial atención a `extractStateDeltas` y `mergeStates` — la división de estado en tres niveles `app:`/`user:`/sesión que explica el README de este módulo, reimplementada desde cero porque la interfaz `session.Service` de Go no te da lógica por defecto en la cual apoyarte.

### Paso 3: Verifica Contra la Propia Suite de Conformidad del SDK ✅

```bash
go test ./internal/infrastructure/redissession/... -v
```

Salida real y confirmada de este comando exacto (abreviada — la corrida completa también registra las líneas propias de arranque de Testcontainers, cada subprueba de `Create`/`Get`/`AppendEvent` individualmente, y tres pruebas más de nivel superior cubriendo `session.State`/`session.Events` directamente):

```
=== RUN   TestRedisSessionService
[...Testcontainers startup logging...]
=== RUN   TestRedisSessionService/Create
=== RUN   TestRedisSessionService/Create/full_key
[...]
=== RUN   TestRedisSessionService/StateManagement/app_state_is_shared
=== RUN   TestRedisSessionService/StateManagement/user_state_is_user_specific
=== RUN   TestRedisSessionService/StateManagement/temp_state_is_not_persisted
--- PASS: TestRedisSessionService (0.60s)
=== RUN   TestAppendEvent_TempStateVisibleWithinSameInvocation
--- PASS: TestAppendEvent_TempStateVisibleWithinSameInvocation (0.37s)
[...]
PASS
ok      github.com/v8tix/adk-training-go/internal/infrastructure/redissession 1.46s
```

`sessiontestsuite.RunServiceTests` levanta un contenedor Redis real vía Testcontainers para toda la corrida y lo vacía entre subpruebas — sin archivos de fixture, sin cliente Redis simulado, todo de verdad.

### Paso 4: Demuestra que la Persistencia Sobrevive un Reinicio de Proceso 🔄

`internal/agents/persistentagent` define un agente mínimo, sin herramientas — esta lección es sobre almacenamiento, no sobre lógica de agentes, así que lo mantenemos simple. `cmd/persistent-agent` inyecta el servicio respaldado por Redis en un `runner.Runner` real:

```go
sessionService := redissession.NewService(redisClient)
r, err := runner.New(runner.Config{
    AppName:        "extensibility_demo",
    Agent:          rootAgent,
    SessionService: sessionService,
    AutoCreateSession: true,
})
```

Córrelo una vez para guardar el dato, en un proceso:

```bash
go run ./cmd/persistent-agent set
```

**Salida real capturada:**

```
🔥 persistent-agent using Redis at localhost:6379
Got it! I'll remember that your favorite color is blue. 💙 Let me know if there's anything I can help you with today!
```

Ahora detenlo por completo, y córrelo *de nuevo* — un proceso genuinamente distinto — preguntando por el dato:

```bash
go run ./cmd/persistent-agent ask
```

**Salida real capturada:**

```
🔥 persistent-agent using Redis at localhost:6379
Your favorite color is **blue**! 💙
```

El segundo proceso nunca compartió memoria con el primero — solo el mismo Redis. Ahí está la prueba. 🎉

`persistent-agent set` también acepta un color opcional, por ejemplo `go run ./cmd/persistent-agent set teal` — `ask` no toma argumentos (la pregunta de recuerdo está fija por diseño). El propio `parseArgs` de `cmd/persistent-agent/main.go` es lo que hace cumplir esto: un bug real, confirmado en vivo, hacía que la CLI original descartara silenciosamente cualquier argumento más allá de `set`/`ask` (así que `persistent-agent set "My favorite color is teal."` igual establecía "blue", sin ningún error) — corregido para realmente usar un color personalizado cuando se le da uno, y para rechazar un argumento inesperado en `ask` con un error claro en vez de ignorarlo.

### Troubleshooting

Ve [troubleshooting.md](./troubleshooting.md) si algún paso no se comporta como esperabas.

### Resumen del Laboratorio 🎉

Implementaste un `session.Service` personalizado desde cero — incluyendo su lógica de división de estado en tres niveles `app:`/`user:`/sesión — lo verificaste contra la suite de conformidad oficial del SDK, y demostraste que la persistencia sobrevive un reinicio de proceso real. ¡Excelente trabajo!

### Preguntas de Autorreflexión 🤔
- `session.Service` no tiene cuerpos de método por defecto en los cuales apoyarte. ¿Qué tendrías que hacer bien tú mismo si estuvieras implementando un cuarto alcance — digamos, un prefijo `global:` compartido entre todas las apps?
- `sessiontestsuite.RunServiceTests` detectó un bug real (el alcance de estado) que una prueba manual y ad hoc podría haber pasado por alto por completo. ¿Qué hizo eso posible?
- Si quisieras agregar un segundo backend (digamos, Postgres) junto a Redis, ¿qué tendrías que cambiar en `cmd/persistent-agent/main.go`?

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python escribe `firestore_provider.py` contra Firestore real de GCP y llama manualmente a `super().append_event(...)` para obtener el manejo por defecto de state-delta. Este laboratorio escribe el equivalente en Go contra Redis, sin ningún `super()` que llamar — la lógica de alcance de estado (`extractStateDeltas`/`mergeStates` en `service.go`) se reimplementa por completo, y luego se verifica con la suite de conformidad propia del SDK en vez de un chequeo manual de "córrelo y mira."
