# Módulo 25: Observabilidad Avanzada con Plugins (Go) 🔭📊

## Teoría

### Observando un Agente Sin Tocar su Lógica

Una herramienta que puede fallar es solo la mitad de la historia — también necesitas saber *cuándo* falló, con qué frecuencia, y si el agente se recuperó. Meter esa lógica directamente en cada función de herramienta funciona, pero llena la lógica de negocio de preocupaciones transversales y hay que repetirla por herramienta. Un **Plugin** resuelve esto sentándose a nivel del Runner, observando cada llamada a herramienta y cada evento a través de todo el agente, sin que ni la herramienta ni el agente necesiten saber que existe.

### Un Plugin Es un Conjunto de Callbacks, No una Clase para Heredar

`google.golang.org/adk/v2/plugin` construye un plugin a partir de una configuración con campos de callback nombrados — sin interfaz que implementar, sin tipo base que embeber:

```go
func newAlertingPlugin(name string) (*plugin.Plugin, *alertTracker, error) {
	tracker := &alertTracker{}
	return plugin.New(plugin.Config{
		Name:                name,
		OnToolErrorCallback: tracker.onToolError,
		OnEventCallback:     tracker.onEvent,
	})
}
```

`OnToolErrorCallback` se dispara cuando una herramienta devuelve un error genuino — no una falla "suave" estructurada como `{"status": "error"}`, sino un `error` de Go real. Devolver un resultado no nulo desde el callback (en vez del error) le dice al framework "ya manejé esto, deja que el agente se recupere con gracia" — exactamente el mismo contrato de recuperación del que depende la herramienta `riskyOperation` de este módulo:

```go
func (a *alertTracker) onToolError(_ agent.Context, t tool.Tool, _ map[string]any, err error) (map[string]any, error) {
	a.hadErrorThisTurn = true
	a.errorCount++
	// ... registra una alerta, escalando pasado un umbral ...
	return map[string]any{"status": "error", "message": err.Error()}, nil
}
```

`OnEventCallback` se dispara en cada evento que produce el agente; revisar `event.IsFinalResponse()` — el mismo método que los tests de todo módulo anterior han usado para encontrar la respuesta real de un turno — le permite a un plugin distinguir "este turno acaba de terminar" de "este es un paso intermedio", así que `alertTracker` solo reinicia su contador una vez que un turno termina limpio.

### Registrado en el Runner, No en el Agente

Un Plugin se mantiene deliberadamente separado de los propios espacios de callback de un agente (`BeforeToolCallbacks`/`AfterToolCallbacks`/`OnToolErrorCallbacks` de `llmagent.Config`) — esos son por agente, esto es transversal. Se registra vía `runner.Config.PluginConfig`/`launcher.Config.PluginConfig`, que tanto el sub-launcher `console` como el `web` ya conectan directamente al `Runner` real:

```go
config := &launcher.Config{
	AgentLoader:  agent.NewSingleLoader(rootAgent),
	PluginConfig: runner.PluginConfig{Plugins: []*plugin.Plugin{alertingPlugin}},
}
```

### Cada Nodo Ya Tiene Trazado Real, Consciente del Grafo

Más allá de los plugins personalizados, el SDK trae soporte nativo para OpenTelemetry conectado — y es genuinamente consciente del grafo, no un envoltorio genérico. `internal/telemetry/node_tracing.go` inicia un span real llamado `invoke_agent <name>` (o `invoke_workflow`/`invoke_node` para los propios nodos de un grafo), etiquetado con un atributo `gen_ai.agent.name` — confirmado en vivo esta sesión capturando un span real con un exportador en memoria y leyendo su nombre y atributos directamente. Su propio comentario de documentación indica que deliberadamente refleja las convenciones semánticas del propio módulo `node_tracing` de `adk-python`.

### Ya Está Conectado en Cada Programa `cmd/`

Tanto el launcher `console` como el `web` exponen una bandera `-otel_to_cloud`, conectada directamente a `google.golang.org/adk/v2/telemetry`, sin necesitar ningún cambio de código:

```bash
go run ./cmd/observability-agent console -otel_to_cloud=false   # default: no necesita credenciales de nube
go run ./cmd/observability-agent console -otel_to_cloud=true    # necesita Application Default Credentials reales de Google
```

El propio laboratorio y tests de este módulo solo usan el default seguro — probando que el mecanismo funciona mediante una exportación OTLP real a un contenedor Jaeger local y desechable, en vez de un proyecto real de Google Cloud.

### Un Problema Real #1: el Proveedor Global se Fija Una Sola Vez por Proceso

`SetGlobalOtelProviders()` registra tu `*telemetry.Providers` en el registro global de OTel — pero el propio código emisor de spans del SDK (`internal/telemetry`) no consulta ese registro de nuevo en cada span. Resuelve un handle de `Tracer` desde el registro global *una sola vez*, hacia una variable a nivel de paquete, la primera vez que cualquier código instala un proveedor real en el proceso. La API de Go de OTel tiene un mecanismo delegante justamente para que un handle obtenido antes de que exista un proveedor real siga funcionando una vez que se instale uno — pero ese delegado se fija *una sola vez*: una *segunda* llamada a `SetGlobalOtelProviders()` más tarde en el mismo proceso no redirige un handle ya fijado. Construir el propio test de telemetría de este módulo topó con esto directamente: dividir la verificación del "exportador en memoria" y la del "export real a Jaeger" en dos funciones de test de nivel superior separadas, cada una con su propia configuración de `telemetry.New`/`SetGlobalOtelProviders()`, rompió en silencio la segunda — sus spans nunca llegaron a su propio exportador. La solución: un proceso obtiene una sola llamada real de instalación de proveedor. Si necesitas varios exportadores, conéctalos todos al *mismo* proveedor antes de instalarlo, no a dos proveedores instalados uno después del otro.

### Un Problema Real #2, Confirmado al Encontrarlo de Verdad

Construir el propio test de telemetría de este módulo sacó a la luz un segundo detalle que vale la pena saber: el método `Shutdown` de `go.opentelemetry.io/otel/sdk/trace/tracetest.InMemoryExporter` llama a `Reset()` internamente — borra en silencio cada span que tiene guardado. Llamar a `Shutdown` sobre un `*telemetry.Providers` compartido para vaciar un exportador *distinto* (uno OTLP real, en este caso) también borrará un exportador en memoria adjunto al mismo proveedor, si lees sus spans después. La solución: lee lo que necesites de un exportador en memoria *antes* de llamar a `Shutdown`, no después — confirmado de la forma difícil, no asumido.

### Puntos Clave ✅
- Un Plugin observa las llamadas a herramientas y eventos reales de un agente sin tocar el código de ninguno de los dos — registrado a nivel del Runner/launcher, no por agente.
- Una instancia de Plugin se comparte durante toda la vida del proceso una vez registrada — protege cualquier estado mutable que mantenga (como un contador de errores en curso) con un mutex, ya que el SDK corre las múltiples llamadas a herramientas de un turno de forma concurrente por defecto, y un launcher multiusuario comparte un mismo plugin entre todas las sesiones.
- `OnToolErrorCallback` se dispara solo ante un `error` de Go genuino; devolver un resultado no nulo en su lugar deja que el agente se recupere con gracia.
- `OnEventCallback` + `event.IsFinalResponse()` es cómo un plugin distingue un turno completado de un paso intermedio.
- El trazado OTel consciente del grafo es real y ya está corriendo — spans `invoke_agent`/`invoke_workflow`/`invoke_node` con atributos `gen_ai.*` reales, confirmado en vivo vía un span capturado, no solo citado de la documentación.
- Cada programa `cmd/` ya tiene una bandera `-otel_to_cloud`; el default seguro no necesita ninguna credencial de nube.
- Un proveedor OTel real solo se fija una sola vez por proceso — instala cada exportador que necesites en un mismo proveedor, no en dos proveedores instalados uno después del otro.
- `tracetest.InMemoryExporter.Shutdown()` borra sus propios spans registrados — léelos antes de apagar, no después.

<hr/>

> **¿Vienes de Python?** 🐍 El propio módulo de Python cubre el mismo Sistema de Plugins (`BasePlugin`, `on_tool_error_callback`, `on_event_callback`, `App(plugins=[...])`) y la misma integración nativa OTel/Cloud Trace (`get_gcp_exporters`, `maybe_set_otel_providers`, cada evento llevando un campo `node_info`). La versión de Go coincide de cerca en espíritu — una configuración de funciones de callback en vez de una `BasePlugin` heredada, el mismo contrato de recuperación, el mismo objetivo de trazado consciente del grafo — solo que expresado como atributos de span OTel reales en vez de un objeto `node_info` copiado sobre el propio evento.
