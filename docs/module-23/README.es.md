# Módulo 23: Manejo de Archivos con Artifacts (Go) 📄🖼️

## Teoría

### Por Qué Artifacts, No Solo Estado

El estado de sesión del Módulo 22 es genial para datos pequeños clave-valor — una preferencia, un puntaje acumulado. Pero un reporte real, un gráfico generado, o un documento procesado es un *archivo*, y merece su propia historia: cada vez que lo guardas, quieres conservar la versión anterior, no sobrescribirla en silencio. Eso es exactamente lo que da el sistema de **Artifacts** de Go.

### Un Solo Almacén, Versiones Reales

Un artifact es un archivo con nombre, versionado automáticamente en cada guardado. `artifact.Service` (confirmado vía `go doc`) es la interfaz: `Save`, `Load`, `Delete`, `List`, `Versions`, `GetArtifactVersion`. `artifact.InMemoryService()` es la implementación para desarrollo local — el mismo rol que `session.InMemoryService()` del Módulo 22, solo que para archivos. Un `artifact/gcsartifact.NewService(ctx, bucketName, opts...)` real y listo para producción respalda los artifacts con un bucket de Google Cloud Storage cuando necesitas que sobrevivan más allá de un solo proceso.

**Importante, confirmado rastreando la propia implementación de `Save` de ambos backends:** el *primer* guardado de un archivo es la versión **1**, no la versión 0. Tanto `InMemoryService` como el `NewService` de `gcsartifact` calculan `nextVersion := 1` cuando nada existe todavía — los propios tests de este repo lo prueban en vivo (`TestArtifactService_VersionsStartAtOne`), no solo lo citan.

### Un Solo Contexto, Mismo Patrón que el Módulo 22

Igual que `agent.Context.State()` unificó el `ToolContext`/`CallbackContext` de Python para datos de sesión, `agent.Context.Artifacts()` es el *único* accesor que usa cada herramienta de función personalizada para archivos:

```go
func extractText(ctx agent.Context, args ExtractTextArgs) (ExtractTextResult, error) {
    resp, err := ctx.Artifacts().Save(ctx, name, genai.NewPartFromText(content))
    if err != nil {
        return ExtractTextResult{}, err
    }
    return ExtractTextResult{Version: int(resp.Version)}, nil
}
```

`Save(ctx, name, *genai.Part)` devuelve el nuevo número de versión. `Load(ctx, name)` trae la última versión (o una específica vía `LoadVersion`). `List(ctx)` devuelve cada nombre de archivo en el alcance actual. Cada llamada tiene alcance automático a la app/usuario/sesión actual — nunca los pasas explícitamente, de la misma forma que el `ctx.State()` del Módulo 22 tampoco los necesitaba.

### El Contenido Binario Necesita un MIME Type Real

El contenido de texto usa `genai.NewPartFromText(text)`. El contenido binario — una imagen, un PDF, audio — usa `genai.NewPartFromBytes(data, mimeType)`, y el MIME type no es decoración opcional: es cómo cualquier lector posterior (incluyendo la propia herramienta `create_report` de este módulo) distingue un artifact de texto de uno de imagen, vía `part.InlineData.MIMEType`.

```go
resp, err := ctx.Artifacts().Save(ctx, "chart.png", genai.NewPartFromBytes(pngBytes, "image/png"))
```

### Dos Alcances: Sesión y Usuario

Por defecto, un artifact vive solo en la sesión que lo creó. Prefija el nombre de archivo con `user:` y se vuelve visible para ese usuario en *cada* sesión que inicie — confirmado de forma idéntica en el código fuente de ambos backends (`strings.HasPrefix(filename, "user:")` en `artifact/inmemory.go` y `artifact/gcsartifact/service.go`), la misma convención que el Módulo 22 ya estableció para las claves de estado, solo que aplicada a nombres de archivo en vez de claves.

- `"report.txt"` → solo esta sesión.
- `"user:preferences.json"` → este usuario, cualquier sesión.

Los propios tests de este repo prueban ambas direcciones en vivo: `TestArtifactService_UserPrefixScopesAcrossSessions` confirma que un archivo con prefijo `user:` cruza sesiones, y `TestArtifactService_PlainFilenameIsSessionScoped` confirma que uno sin prefijo genuinamente no lo hace (un caso negativo real, no solo el positivo).

### Los Artifacts Faltantes Usan un Centinela de la Librería Estándar

Carga un nombre de archivo que nunca se guardó y `Load` devuelve `fmt.Errorf("artifact not found: %w", fs.ErrNotExist)` — envolviendo el propio `io/fs.ErrNotExist` *de la librería estándar*, no un centinela hecho a medida. Revísalo de la misma forma que revisarías cualquier error envuelto:

```go
if _, err := ctx.Artifacts().Load(ctx, name); errors.Is(err, fs.ErrNotExist) {
    // maneja el caso de que aún no se haya creado
}
```

### Sin Async/Await Que Enseñar

Cada llamada a un artifact aquí es una función de Go ordinaria y síncrona que toma un `context.Context` — no hay una distinción separada de "esto debe esperarse" que aprender. Simplemente no es una dimensión que tenga el modelo de llamadas a herramientas de Go.

### Un Mecanismo Real, con Forma Distinta, para las Credenciales

No hay un método `SaveCredential`/`LoadCredential` en `agent.Context` — confirmado leyendo su conjunto completo de métodos. Pero `google.golang.org/adk/v2/auth` es real y sustancial: un `CredentialProvider` (`StaticToken`, `APIKey`, `ADC`, `ServiceAccount`) resuelve una `Credential` que se escribe a sí misma en una solicitud HTTP saliente, aplicada por llamada vía un `Transport`. Resuelve el mismo problema real — una herramienta necesita un secreto para llamar a algo — solo que con forma de un resolutor a nivel de transporte HTTP en vez de un almacén de secretos con alcance de sesión. El propio laboratorio de este módulo no lo construye de forma práctica, igualando el alcance del propio currículo fuente para este tema.

> **Yendo Más Allá:** si quieres probar el paquete `auth` real, conecta un `CredentialProvider` (`auth.StaticToken` es el punto de partida más simple) al cliente HTTP que use alguna de tus propias herramientas, y confirma que la solicitud saliente realmente lleva la credencial resuelta.

### Puntos Clave ✅
- Un artifact es un archivo con nombre, versionado automáticamente — `Save`/`Load`/`Delete`/`List`/`Versions` de `artifact.Service`.
- El primer guardado de cualquier archivo es la **versión 1**, confirmado en vivo contra el código fuente de ambos backends, el de memoria y el de GCS.
- `agent.Context.Artifacts()` es el único accesor que usa cada herramienta — sin un tipo de contexto separado, igualando el propio patrón `State()` del Módulo 22.
- El contenido binario necesita un MIME type real (`genai.NewPartFromBytes`); un artifact faltante se reporta como el propio `fs.ErrNotExist` de la librería estándar, envuelto.
- El prefijo `user:` en el nombre de archivo da alcance a un artifact a través de cada sesión de ese usuario — probado en vivo en ambas direcciones, positiva y negativa.
- El mecanismo de credenciales real y con forma distinta de Go vive en el paquete `auth` — no es un equivalente uno-a-uno de un almacén de secretos con alcance de sesión, pero tampoco es una ausencia.

<hr/>

> **¿Vienes de Python?** 🐍 El módulo de Python cubre el mismo modelo de Artifact — `save_artifact`/`load_artifact`/`list_artifacts`, `types.Part.from_bytes()`/`from_text()`, la convención de alcance `user:`, `InMemoryArtifactService`/`GcsArtifactService`. Lo único que vale la pena señalar directamente: la documentación de Python describe las versiones como indexadas desde 0 ("el primer guardado crea la versión 0"); los propios backends de Go — ambos, rastreados directamente — empiezan desde 1.
