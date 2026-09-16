# Módulo 13.5: Extendiendo ADK — Persistencia Personalizada con Redis (Go) 🗄️

*Este módulo va junto al módulo 13, no después de los módulos de state/memory, a propósito: escribir un `session.Service` personalizado es en sí mismo un acto de extender el toolkit del ADK, no una preocupación operativa de producción. Piénsalo como "construir un nuevo tipo de herramienta" — solo que esta guarda estado en vez de ejecutar acciones.*

## Teoría

### `session.Service` Es una Interfaz Conectable 🔌

Cada módulo anterior usó `session.InMemoryService()` sin pensarlo dos veces. Resulta que es solo una implementación de una interfaz Go simple:

```go
type Service interface {
    Create(context.Context, *CreateRequest) (*CreateResponse, error)
    Get(context.Context, *GetRequest) (*GetResponse, error)
    List(context.Context, *ListRequest) (*ListResponse, error)
    Delete(context.Context, *DeleteRequest) error
    AppendEvent(context.Context, Session, *Event) error
}
```

A `runner.Runner` no le importa qué implementación le des — inyecta la tuya vía `runner.Config`:

```go
sessionService := redissession.NewService(redisClient)
r, err := runner.New(runner.Config{
    AppName:        "extensibility_demo",
    Agent:          rootAgent,
    SessionService: sessionService, // instead of session.InMemoryService()
})
```

De acá en adelante, cada llamada a `r.Run(...)` persiste a través de tu almacenamiento — sin cambiar ni una línea de instrucciones del agente ni del código de herramientas. 🎉

### El Contrato No Tiene Implementación por Defecto ⚠️

`session.Service` es una interfaz Go simple — implementarla significa escribir tú mismo el comportamiento completo de cada método, incluyendo cómo se aplica un `StateDelta` al estado. Confirmado leyendo directamente el `InMemoryService` propio del SDK: esa aplicación no es una simple copia. Una clave de `StateDelta` se alcanza de tres formas:

- `app:key` — compartida entre **todas las sesiones de la app**, sin importar el usuario
- `user:key` — compartida entre todas las sesiones de **un usuario**, dentro de la app
- cualquier otra clave — privada de esa sesión
- `temp:key` — se aplica al estado de la sesión por el resto de la invocación *actual*, pero **nunca se escribe a almacenamiento durable** — un `Get` posterior nunca la ve

Un `session.Service` personalizado tiene que implementar este alcance él mismo — no hay un valor por defecto compartido al cual recurrir. El `AppendEvent` de `internal/infrastructure/redissession` hace exactamente eso:

```go
appDelta, userDelta, sessionDelta := extractStateDeltas(event.Actions.StateDelta)
mergeHash(ctx, appStateKey(appName), appDelta)           // shared app-wide
mergeHash(ctx, userStateKey(appName, userID), userDelta) // shared per-user
maps.Copy(stored.State, sessionDelta)                     // private to this session
```

### Probando Corrección Contra la Propia Suite de Pruebas del SDK ✅

El laboratorio de Python verifica su proveedor de Firestore corriéndolo una vez y revisando el resultado a ojo. Go tiene algo mucho más fuerte: `session/sessiontestsuite`, una suite de conformidad real, agnóstica al backend, que el SDK trae y usa para probar su *propio* servicio integrado con SQL:

```go
sessiontestsuite.RunServiceTests(t, sessiontestsuite.SuiteOptions{
    SupportsUserProvidedSessionID: true,
}, func(t *testing.T) session.Service {
    client.FlushDB(ctx)
    return redissession.NewService(client)
})
```

Y aquí está la prueba: esto fue lo que realmente detectó el alcance de estado en tres niveles de arriba, en vivo, mientras construíamos este módulo — una versión anterior trataba todo el estado como privado de sesión, y `RunServiceTests` falló dos subpruebas (`app_state_is_shared`, `user_state_is_user_specific`) con un diff preciso y accionable. No una suposición. No una corazonada. Un diff real señalando justo el bug. 🎯

### Un Backend Real, Verificado con Testcontainers 🐳

El laboratorio de Python requiere un proyecto real de Google Cloud y `gcloud auth application-default login`. Este curso nunca antes necesitó una cuenta cloud completa, así que este módulo usa **Redis** en su lugar — nombrado explícitamente en la propia sección de Teoría de Python como una alternativa válida justo para este caso ("tiempos de respuesta casi instantáneos que una base de datos persistente estándar podría no ofrecer"). Las pruebas de `internal/infrastructure/redissession` usan [Testcontainers](https://testcontainers.com/) para levantar un contenedor Redis real y efímero — sin base de datos de prueba manejada a mano, sin mocks reemplazando lo real, y nada corriendo cuando termina la suite de pruebas.

### Puntos Clave ✅
- `session.Service` es una interfaz simple, inyectada en `runner.Config.SessionService` — cualquier implementación funciona, sin cambios en el código de agentes o herramientas.
- La interfaz de Go no tiene cuerpos de método por defecto para heredar — una implementación personalizada es dueña del contrato completo, incluyendo la división de estado en tres niveles `app:`/`user:`/sesión.
- `session/sessiontestsuite.RunServiceTests` verifica una implementación personalizada contra la misma suite que el SDK usa en su propio servicio integrado — una historia de corrección mucho más fuerte que revisar a mano.
- Testcontainers provee un backend real y efímero para pruebas de integración — el primer módulo de este curso que lo necesita.

<hr/>

> **¿Vienes de Python?** 🐍 `BaseSessionService` de Python y `session.Service` de Go cumplen el mismo rol — una interfaz de persistencia conectable, inyectada en el `Runner`. La diferencia real está en la herencia: la clase base abstracta de Python le da a una subclase lógica por defecto real vía `super()`; la interfaz de Go no le da a una implementación personalizada nada más que un conjunto de métodos que satisfacer. El laboratorio de Python también elige Firestore específicamente — este módulo elige Redis en su lugar, una de las alternativas que la propia sección de Teoría de Python nombra, para mantener intacto el requisito de este curso de no necesitar cuenta cloud.
