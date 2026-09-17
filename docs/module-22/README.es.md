# Módulo 22: Estado y Memoria — Contexto Persistente del Agente (Go) 🧠📚

## Teoría

### Un Solo Almacén, Cuatro Alcances

Conociste `agent.Context.State()` en el Módulo 10 para recordar un solo nombre. Este módulo trata de usarlo con más criterio — porque no todo dato debería vivir la misma cantidad de tiempo ni ser visible para la misma audiencia. El SDK traza esa línea con un prefijo en la clave, y hay exactamente cuatro alcances:

| Prefijo | Constante | Vive mientras... | Visible para... |
|---|---|---|---|
| *(ninguno)* | — | Esta sesión | Solo esta conversación |
| `user:` | `session.KeyPrefixUser` | Cada sesión que este usuario tenga siempre | Este usuario, en todas sus sesiones |
| `app:` | `session.KeyPrefixApp` | Para siempre, hasta que cambie | Todos los usuarios, todas las sesiones |
| `temp:` | `session.KeyPrefixTemp` | Solo esta invocación | Nadie después del turno actual |

Leer o escribir cualquiera de ellos es exactamente la misma llamada — `ctx.State().Get(key)`/`ctx.State().Set(key, value)` — el prefijo es solo texto incrustado en la propia clave. No hay una API separada por alcance. 🎯

### Un Solo Contexto, No Dos

La documentación de Python traza una línea entre `ToolContext` (para herramientas) y `CallbackContext` (para callbacks). Go no necesita esa distinción — `agent.Context.State()` es el *único* accesor que recibe cada herramienta de función personalizada, establecido desde el Módulo 9. Leyendo directamente el propio `callbackContextState.Set` del SDK se confirma que hace dos cosas a la vez en cada escritura: actualiza el `EventActions.StateDelta` del evento *y* el estado de sesión en vivo directamente — la misma disciplina de "siempre pasar por el contexto, nunca tocar a mano una sesión ya obtenida" que la documentación de Python exige, solo que colapsada en un solo tipo en vez de dos.

### `temp:` Realmente Desaparece

Vale la pena dudar de esto hasta comprobarlo, así que el propio test de este módulo no se conforma con el comentario de la documentación — lo demuestra. Una herramienta escribe `temp:percentage` a mitad de turno; más tarde, en ese *mismo* turno, se puede leer. Pero una vez que esa llamada a `Run()` termina y arranca una nueva llamada, separada, `session.State().Get("temp:percentage")` devuelve `session.ErrStateKeyNotExist` — se fue, confirmado estructuralmente, no asumido. Las claves `user:`, mientras tanto, escritas en ese mismo turno, siguen ahí esperando. Ese contraste — un prefijo sobrevive, el otro no — es todo el sistema de alcances en un solo experimento.

### Marcadores Opcionales en las Instrucciones: `{key?}`

El string de instrucción de un agente puede traer estado en vivo directo al prompt con `{key}` — pero ¿qué pasa si esa clave nunca se estableció? Un simple `{app:course_version?}` (nota el `?` al final) le dice al SDK "inyéctala si está, sáltala en silencio si no" en vez de fallar en un deploy que nunca alcanzó a poblar esa clave `app:`. Confirmado directamente en `internal/llminternal/instruction_processor.go`: `InjectSessionState` revisa explícitamente ese `?` al final y trata una clave faltante como "déjala en blanco", no como "falla".

### Recuerdo a Largo Plazo y Buscable: `memory.Service`

El estado de sesión responde "qué sabe *esta* conversación." Un `memory.Service` responde una pregunta distinta: "¿alguna conversación *pasada* con este usuario tocó este tema?" Dos métodos hacen todo el trabajo — `AddSessionToMemory(ctx, session)` ingiere una sesión terminada, `SearchMemory(ctx, *SearchRequest)` devuelve entradas coincidentes después, ordenadas por cuántas palabras distintas de la búsqueda coincidieron. `memory.InMemoryService()` es la implementación real, funcional, en proceso, que los propios tests de este módulo ejercitan directamente — sin mocks, sin necesitar ninguna llamada a un LLM para probar que funciona.

### Puntos Clave ✅
- Cuatro alcances de estado, una sola API — el prefijo en el string de la clave es lo único que cambia.
- `agent.Context.State()` unifica lo que Python separa en `ToolContext`/`CallbackContext`, y cada `Set` escribe tanto el delta del evento como el estado en vivo a la vez.
- El estado `temp:` realmente desaparece en la siguiente llamada separada a `Run()` — probado con una lectura directa al servicio de sesión, no inferido de un comentario de documentación.
- `{key?}` deja que una instrucción referencie estado que quizás no exista aún, sin fallar.
- `memory.Service`/`InMemoryService` da recuerdo real entre sesiones — `AddSessionToMemory` y luego `SearchMemory`, ambos demostrados en vivo en los propios tests de este módulo.

<hr/>

> **¿Vienes de Python?** 🐍 El módulo de Python cubre los mismos cuatro prefijos, la misma separación `ToolContext`/`CallbackContext` (unificada aquí en un solo `agent.Context`), el mismo templating opcional `{key?}`, y la misma forma de `MemoryService` (`add_session_to_memory`/`search_memory` → `AddSessionToMemory`/`SearchMemory`). Lo único que vale la pena señalar directamente: el `.get(key, default)` de Python toma su valor de respaldo inline; Go expresa "no encontrado" como el centinela explícito `session.ErrStateKeyNotExist` que tu propio código chequea, el mismo patrón que el Módulo 10 ya estableció.
