# Solución de Problemas: Módulo 20 (Go) 🔧

### La respuesta real del loop no es el último mensaje de chat visible 🤔

**Síntoma:** tu código (un test, un visor de trazas propio) lee "el último evento con texto" o "la última respuesta final", esperando que sea el resultado refinado — y obtiene otra cosa completamente distinta. En el caso propio de este laboratorio, esa "otra cosa" fue la respuesta final del crítico, `"APPROVED"`.

**Causa:** confirmado en vivo — el evento terminal del propio nodo dinámico (el valor real de retorno del loop) es un evento *separado* de cualquiera de los turnos de chat del writer/critic/refiner. Su autor es el nombre del propio workflow agent raíz (`"EssayRefiner"` en este laboratorio), con `Content: nil` y `Output` con el resultado real. El evento de contenido de chat que resulte quedar de último en una corrida dada (eso depende de cuántas iteraciones corrieron) *no* es tu respuesta.

**Solución:** lee `event.Output` específicamente del evento cuyo `Author` coincida con el nombre de tu workflow — no "lo que quedó de último". Revisa `agent_test.go` para ver el patrón que funciona.

### Un test que verifica contenido exacto del ensayo pasa en un backend pero falla en otro 😬

**Síntoma:** un test que verifica que la historia final contenga una palabra o frase específica pasa siempre en Gemini pero falla de forma intermitente (o con una historia claramente relacionada pero no literal) en Ollama local.

**Causa:** confirmado en vivo en este módulo — una versión anterior de la instrucción del crítico en este laboratorio pedía un "tesoro escondido" como *concepto*, y Ollama aprobaba felizmente una historia que satisfacía el concepto de forma temática (una caja escondida de monedas y una carta vieja) sin usar nunca la palabra literal "treasure". Gemini sí usaba la palabra literal por casualidad; Ollama no. El criterio de aprobación del crítico era semántico, pero la verificación del test era una coincidencia literal de string — un desajuste entre lo que el crítico realmente exige y lo que el test realmente verifica.

**Solución:** haz que la instrucción del propio crítico exija algo literal y verificable ("la historia debe contener la palabra 'treasure'" — no "la historia debe mencionar un tesoro escondido"), para que el criterio de aprobación del modelo y la verificación del test comprueben *exactamente* lo mismo. La instrucción `critic_instruction.md` de este laboratorio se reescribió así en cuanto detectamos el desajuste en vivo.
