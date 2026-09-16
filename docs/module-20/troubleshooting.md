# Troubleshooting: Module 20 (Go) 🔧

### The loop's actual answer isn't the last visible chat message 🤔

**Symptom:** your code (a test, a custom trace viewer) reads "the last event with text" or "the last final response," expecting it to be the refined result — and gets something else entirely. In this lab's own case, that "something else" was the critic's own `"APPROVED"` reply.

**Cause:** confirmed live — the dynamic node's own terminal event (the loop's real return value) is a *separate* event from any of the writer/critic/refiner's own chat turns. It's authored by the root workflow agent's own name (`"EssayRefiner"` in this lab), with `Content: nil` and `Output` set to the actual result. Whichever chat-content event happens to land last in a given run (that depends on how many iterations ran) is *not* your answer.

**Fix:** read `event.Output` specifically from the event whose `Author` matches your workflow's own name — not "whatever came last." Check `agent_test.go` for the working pattern.

### A test asserting on exact essay content passes on one backend but fails on another 😬

**Symptom:** a test checking the final story for a specific word or phrase passes reliably on Gemini but fails intermittently (or with a clearly-related-but-not-literal story) on local Ollama.

**Cause:** confirmed live this module — an earlier version of this lab's own critic instruction asked for a "hidden treasure" as a *concept*, and Ollama happily approved a story that satisfied the concept thematically (a hidden box of coins and an old letter) without ever using the literal word "treasure." Gemini happened to use the literal word anyway; Ollama didn't. The critic's own approval judgment was semantic, but the test's assertion was a literal string match — a mismatch between what the critic actually enforces and what the test actually checks.

**Fix:** make the critic's own instruction require something literal and checkable ("the story must contain the word 'treasure'" — not "the story must mention a hidden treasure"), so the model's own approval criterion and the test's assertion are checking the *exact* same thing. This lab's own `critic_instruction.md` got rewritten this way once we caught the mismatch live.
