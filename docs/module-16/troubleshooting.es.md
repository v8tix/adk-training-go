# Solución de Problemas: Módulo 16 (Go) 🛠️

### El modelo local se niega a inventar titulares 🤷

**Causa:** ninguno de los agentes investigadores tiene una herramienta de búsqueda real, así que un modelo local de Ollama al que se le pide "buscar titulares" sin nada que buscar puede honestamente declinar en vez de fabricar una respuesta que suene plausible — un comportamiento real y confirmado, ¡no un bug!

**Solución:** cambia a `MODEL_TYPE=gemini` si quieres un ejemplo más completo — Gemini tiende a responder con confianza desde su propio entrenamiento en vez de declinar. Mira el README de este módulo para la diferencia completa y confirmada entre backends.

### La instrucción del summarizer muestra literalmente `{tech_news}` en vez de contenido real 👀

**Causa:** el `OutputKey` configurado en un agente investigador no coincide exactamente con el nombre del placeholder usado en `summarizer_instruction.md` — la sustitución se hace por coincidencia exacta de string, no aproximada.

**Solución:** confirma que ambos `OutputKey`s coincidan exactamente con los nombres de placeholder en `summarizer_instruction.md`.
