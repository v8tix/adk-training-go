# Lab 22: Building a Personal Learning Tutor (Go) 🧠📚

## Goal

Build a tutor agent that exercises all four session-state scopes — `user:`, `app:`, plain session, and `temp:` — plus a real, working demonstration of `memory.Service`.

## Lab Tasks

### 1. Read `internal/agents/personaltutor/tools.go`

Six handlers, each showing a different scope in action:

- `setUserPreferences` writes `user:language`/`user:difficulty_level` — persists across every future session for this user.
- `recordTopicCompletion` reads-then-appends `user:topics` (`[]string`) and `user:scores` (`map[string]int`), defaulting both to empty the first time via `errors.Is(err, session.ErrStateKeyNotExist)`.
- `getUserProgress` reads all the `user:*` keys above and computes an average — a first-time user with nothing stored yet gets zero/default values back, not an error.
- `startLearningSession` writes a plain, unprefixed `current_topic` — scoped to just this session — while reading `user:difficulty_level` for personalization.
- `calculateQuizGrade` writes `temp:percentage`/`temp:raw_score` — intermediate values nobody needs after this one call — and returns a letter grade.
- `searchPastLessons` does a simple substring search over `user:topics`. (In production this is exactly where you'd reach for `memory.Service.SearchMemory` instead — see Task 5 for that mechanism proven directly.)

### 2. Read `internal/agents/personaltutor/agent.go` and its prompt

Same `functiontool.New` + `Tools` shape as every prior module's agent. Its instruction (`prompts/tutor_instruction.md`) opens with `Course Version {app:course_version?}` — the trailing `?` means this renders cleanly even though no `app:course_version` key is ever set in this lab.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/personal-tutor console
```

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability; the tutor personalized its own reply language to Spanish once that preference was stored — nobody told it to do that, `user:language` did):

```
📚 personal-tutor using qwen3.8:27b

User -> Please set my preferred language to Spanish and my difficulty level to intermediate.
Agent -> ¡Listo! He guardado tus preferencias:

- **Idioma:** Español
- **Nivel:** Intermedio

A partir de ahora adaptaré mis explicaciones al nivel intermedio y responderé en español. ¿Qué tema te gustaría empezar a estudiar?

User -> I would like to start learning about Goroutines now.
Agent -> ¡Perfecto! Hemos comenzado una sesión de estudio sobre **Goroutines**, adaptada a tu nivel intermedio.
[...concept explanation, trimmed...]

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

Four separate turns, same session — `user:difficulty_level` set in turn 1 personalized turn 2's session start, and `user:topics`/`user:scores` from turn 3 fed straight into turn 4's progress summary. `internal/agents/personaltutor/agent_test.go` proves this structurally, not just by eyeballing the transcript above.

### 4. Read `internal/agents/personaltutor/tools_test.go`

Pure unit tests, no LLM — the same `fakeState`/`fakeContext` double from Module 10, exercised across every scope: `user:` writes/reads, the plain `current_topic` key, and `temp:` writes, plus every missing-key default path.

### 5. Read `internal/agents/personaltutor/agent_test.go`

`TestPersonalTutor_PersistsUserStateAcrossTurns_{Ollama,Gemini}` drives **four** separate `Run()` calls against one session, then reads the session directly (`sessionService.Get`) afterward to check two things structurally: `user:language`, `current_topic`, and `user:topics` all survived every turn, while `temp:percentage`/`temp:raw_score` come back `session.ErrStateKeyNotExist` — gone, exactly as the scope promises.

### 6. Read `internal/agents/personaltutor/memory_test.go`

This lab's own `search_past_lessons` only simulates a search (Task 1). This file proves the real mechanism it's standing in for actually works: build a session with real content, `AddSessionToMemory` it, then `SearchMemory` with a matching query and get a real result back — plus a negative-case test confirming an unrelated query correctly returns nothing.

## Self-Reflection Questions 🤔
- Why does `calculateQuizGrade` use `temp:` instead of `user:` for its intermediate percentage? What would go wrong if it didn't?
- `getUserProgress` never errors for a brand-new user with nothing stored. What pattern makes that possible, and where else in this module does the same pattern show up?
- If you wired `search_past_lessons` to call the real `memory.Service.SearchMemory` instead of simulating the search, what would you need to add to `BuildRootAgent`?
- What would `{app:course_version?}` render as if `app:course_version` *were* set — and where would you set it?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/personaltutor/tools.go` and `agent.go` for the real mechanism — six handlers, each touching a different state scope, wrapped the same way every prior module's tools were.
