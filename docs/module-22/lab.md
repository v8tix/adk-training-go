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

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability). `user:language`/`user:difficulty_level`, once stored in turn 1, personalize every later turn in the same session — turn 2's session start and turn 4's summary both read them straight back from state, not by re-deriving them from the conversation:

```
📚 personal-tutor using qwen3.8:27b

User -> Please set my preferred language to English and my difficulty level to intermediate.
Agent -> Done! Your preferences are now set to English and intermediate level. Whenever you're ready, let me know what topic you'd like to dive into and I'll tailor the material to your level.

User -> I would like to start learning about Goroutines now.
Agent -> Great choice! I've started your Goroutines session at the intermediate level.
[...concept explanation, trimmed...]

User -> I just took the Goroutines quiz and got 8 out of 10 correct. Please grade it, then record my completion of the Goroutines topic with that score.
Agent -> Nice work! Here's your result:

- **Score:** 8/10 → **80%**
- **Grade:** **B**

I've recorded your completion of the Goroutines topic with a score of 80/100.

User -> How is my overall learning progress so far?
Agent -> Here's where you stand so far:

- **Topics completed:** 1 — Goroutines
- **Average quiz score:** 80%
- **Difficulty level:** Intermediate
- **Language:** English
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
