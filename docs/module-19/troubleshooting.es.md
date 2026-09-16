# Solución de Problemas: Módulo 19 (Go) 🛠️

### La respuesta de un sub-agente en modo task no aparece como su propio evento/autor 🤔

**Síntoma:** el código que inspecciona el flujo de eventos (un test, un visor de trazas personalizado) espera que la respuesta de un sub-agente `ModeTask`/`ModeSingleTurn` aparezca como un evento separadamente autorado y final — como hace el agente al que se transfirió en un traspaso `ModeChat` — y encuentra que no está ahí, o encuentra que el propio coordinador aparece acreditado como el autor en su lugar.

**Causa:** confirmado en vivo este módulo — el despacho task/single-turn ocurre a través de una herramienta de función inyectada por el framework (`TaskAgentTool`/`SingleTurnTool`), no `transfer_to_agent`. El agente que llama se mantiene como el que produce la salida visible de la conversación; el resultado del sub-agente es data que consume, no un traspaso de "quién está hablando." El propio texto de un sub-agente puede terminar plegado directamente dentro de la respuesta del coordinador dentro del mismo turno.

**Solución:** no asevere sobre `event.Author` o `event.IsFinalResponse()` para detectar la contribución de un sub-agente task/single-turn. Aseverá sobre el contenido visible real en su lugar — concatena el texto no-thought de cada evento del turno y revisa la información que esperas que esté presente, de la forma en que lo hace el propio `agent_test.go` de este módulo (verificando que el plan final mencione la aerolínea que dio el usuario, en vez de verificar a qué agente "pertenece" cada evento).

### `OutputKey` en un agente en modo task se sobrescribe antes de que la tarea termine ⚠️

**Síntoma:** el código revisa `event.Actions.StateDelta[someOutputKey]` esperando que se pueble solo una vez que el agente en modo task termina (llama a `finish_task`), pero la key ya está presente después de la primera respuesta del agente, todavía incompleta (p. ej. su pregunta aclaratoria).

**Causa:** confirmado en vivo este módulo — `OutputKey` escribe en cada una de las respuestas propias de ese agente, no solo en la terminal. La pregunta a mitad de conversación de un agente en modo task cuenta como "su propia" respuesta para este propósito, igual que su eventual respuesta completa.

**Solución:** no uses la sola presencia de `OutputKey` como señal de "¿esta tarea terminó?". Si necesitas distinguir "todavía preguntando" de "terminado," revisa el contenido real del valor, o reestructura la verificación en torno a comportamiento externamente observable (como en el propio test de este módulo) en vez de la presencia de la key.
