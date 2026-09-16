# Troubleshooting: Módulo 13.5 (Go) 🔧

### `ask` no recuerda lo que guardó `set`

**Causa:** las dos invocaciones no apuntan a la misma instancia de Redis, o no usan el mismo `appName`/`userID`/`sessionID`.

**Solución:** verifica que ambas invocaciones apunten al mismo `REDIS_ADDR` y al mismo `appName`/`userID`/`sessionID` — `cmd/persistent-agent/main.go` los define como constantes fijas para este laboratorio; una aplicación real los derivaría por usuario.

### Las pruebas no logran arrancar un contenedor 🐳

**Causa:** Docker no está corriendo, o Testcontainers no puede alcanzarlo.

**Solución:** verifica que Docker esté corriendo (`docker info`). Las pruebas se saltan, en vez de fallar, si Testcontainers genuinamente no puede alcanzar Docker en absoluto — un error de build o conexión dentro de un Docker que sí está corriendo es un problema distinto y real que vale la pena investigar por separado.
