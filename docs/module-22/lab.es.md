# Laboratorio 22: Construyendo un Tutor de Aprendizaje Personal (Go) 🧠📚

## Objetivo

Construye un agente tutor que ejercite los cuatro alcances de estado de sesión — `user:`, `app:`, sesión simple, y `temp:` — más una demostración real y funcional de `memory.Service`.

## Tareas del Laboratorio

### 1. Lee `internal/agents/personaltutor/tools.go`

Seis handlers, cada uno mostrando un alcance distinto en acción:

- `setUserPreferences` escribe `user:language`/`user:difficulty_level` — persiste en cada sesión futura de este usuario.
- `recordTopicCompletion` lee y luego agrega a `user:topics` (`[]string`) y `user:scores` (`map[string]int`), inicializando ambos vacíos la primera vez vía `errors.Is(err, session.ErrStateKeyNotExist)`.
- `getUserProgress` lee todas las claves `user:*` de arriba y calcula un promedio — un usuario nuevo sin nada guardado recibe valores en cero/default de vuelta, no un error.
- `startLearningSession` escribe un `current_topic` simple, sin prefijo — con alcance solo a esta sesión — mientras lee `user:difficulty_level` para personalizar.
- `calculateQuizGrade` escribe `temp:percentage`/`temp:raw_score` — valores intermedios que nadie necesita después de esta llamada — y devuelve una calificación con letra.
- `searchPastLessons` hace una búsqueda simple por substring sobre `user:topics`. (En producción, es exactamente donde recurrirías a `memory.Service.SearchMemory` en su lugar — ver la Tarea 5 para ver ese mecanismo probado directamente.)

### 2. Lee `internal/agents/personaltutor/agent.go` y su prompt

Misma forma con `functiontool.New` + `Tools` que el agente de todo módulo anterior. Su instrucción (`prompts/tutor_instruction.md`) abre con `Course Version {app:course_version?}` — el `?` al final significa que esto se renderiza limpio aunque nunca se establezca ninguna clave `app:course_version` en este laboratorio.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/personal-tutor console
```

Sin `.env`, sin API key. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad; el tutor personalizó su propia respuesta al español una vez que se guardó esa preferencia — nadie le dijo que hiciera eso, lo hizo `user:language`):

```
📚 personal-tutor using qwen3.8:27b

User -> Please set my preferred language to Spanish and my difficulty level to intermediate.
Agent -> ¡Listo! He guardado tus preferencias:

- **Idioma:** Español
- **Nivel:** Intermedio

A partir de ahora adaptaré mis explicaciones al nivel intermedio y responderé en español. ¿Qué tema te gustaría empezar a estudiar?

User -> I would like to start learning about Goroutines now.
Agent -> ¡Perfecto! Hemos comenzado una sesión de estudio sobre **Goroutines**, adaptada a tu nivel intermedio.
[...explicación del concepto, recortada...]

User -> I just took the Goroutines quiz and got 8 out of 10 correct. Please grade it, then record my completion of the Goroutines topic with that score.
Agent -> ¡Excelente trabajo! Aquí está el resumen de tu quiz de Goroutines:

| Detalle | Resultado |
|---------|-----------|
| Respuestas correctas | 8 / 10 |
| Porcentaje | 80% |
| **Calificación** | **B** |

He registrado la finalización del tema **Goroutines** con una puntuación de **80/100**.

User -> How is my overall learning progress so far?
Agent -> ¡Aquí tienes un resumen de tu progreso actual! 📊

- **Temas completados:** 1
  - Goroutines — **80/100** (B)
- **Promedio de puntuación:** 80%
- **Nivel de dificultad:** Intermedio
- **Idioma de estudio:** Español
```

Cuatro turnos separados, misma sesión — `user:difficulty_level`, establecido en el turno 1, personalizó el inicio de sesión del turno 2, y `user:topics`/`user:scores` del turno 3 alimentaron directamente el resumen de progreso del turno 4. `internal/agents/personaltutor/agent_test.go` prueba esto de forma estructural, no solo mirando la transcripción de arriba.

### 4. Lee `internal/agents/personaltutor/tools_test.go`

Tests unitarios puros, sin LLM — el mismo par `fakeState`/`fakeContext` del Módulo 10, ejercitado a través de cada alcance: escrituras/lecturas `user:`, la clave simple `current_topic`, y escrituras `temp:`, más cada ruta de valor por defecto para clave faltante.

### 5. Lee `internal/agents/personaltutor/agent_test.go`

`TestPersonalTutor_PersistsUserStateAcrossTurns_{Ollama,Gemini}` hace **cuatro** llamadas separadas a `Run()` contra una misma sesión, y luego lee la sesión directamente (`sessionService.Get`) después para chequear dos cosas de forma estructural: que `user:language`, `current_topic`, y `user:topics` sobrevivieron todos los turnos, mientras que `temp:percentage`/`temp:raw_score` vuelven como `session.ErrStateKeyNotExist` — desaparecidos, exactamente como promete el alcance.

### 6. Lee `internal/agents/personaltutor/memory_test.go`

La propia herramienta `search_past_lessons` de este laboratorio solo simula una búsqueda (Tarea 1). Este archivo prueba que el mecanismo real al que reemplaza realmente funciona: construye una sesión con contenido real, hazle `AddSessionToMemory`, y luego hazle `SearchMemory` con una búsqueda que coincida y obtén un resultado real de vuelta — más un test de caso negativo confirmando que una búsqueda no relacionada correctamente no devuelve nada.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `calculateQuizGrade` usa `temp:` en vez de `user:` para su porcentaje intermedio? ¿Qué saldría mal si no lo hiciera?
- `getUserProgress` nunca falla para un usuario nuevo sin nada guardado. ¿Qué patrón hace eso posible, y dónde más en este módulo aparece el mismo patrón?
- Si conectaras `search_past_lessons` para que llamara al `memory.Service.SearchMemory` real en vez de simular la búsqueda, ¿qué necesitarías agregar a `BuildRootAgent`?
- ¿Qué renderizaría `{app:course_version?}` si `app:course_version` *sí* estuviera establecida — y dónde la establecerías?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/personaltutor/tools.go` y `agent.go` para el mecanismo real — seis handlers, cada uno tocando un alcance de estado distinto, envueltos de la misma forma que las herramientas de todo módulo anterior.
