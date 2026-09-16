# Solución de Problemas: Módulo 17 (Go) 🛠️

### La consola muestra el texto de razonamiento crudo del modelo mezclado con la salida de `Agent ->` 🤯

**Causa:** confirmado en vivo en el modelo local de Ollama usado en este curso (`qwen3.8:27b`, un modelo capaz de razonar): su texto de cadena de pensamiento no se filtra antes de que se imprima la salida del nodo, así que el turno del clasificador de `classify_and_route` y el turno de cada especialista pueden mostrar razonamiento visible antes del JSON o análisis real. La salida de Gemini fue limpia en comparación, en la misma corrida — una diferencia real y confirmada entre backends, no un bug en el código de este módulo.

**Solución:** cambia a `MODEL_TYPE=gemini` si quieres una transcripción limpia para fines de demostración. La lógica de enrutamiento en sí no se ve afectada de ninguna forma — `classify_and_route` lee el resultado estructurado `OutputSchema` del clasificador, no su texto visible, así que la fuga de razonamiento en la consola no cambia qué especialista se elige. 👍

### `classify_and_route` devuelve un error como "classify_and_route: classifier returned no currency" ❌

**Causa:** la respuesta restringida por `OutputSchema` del clasificador no incluyó un campo `currency` no vacío — lo más probable es que la cuantización del modelo subyacente no soporte decodificación restringida por JSON-schema (el propio hallazgo confirmado del módulo 4 para *algunas* cuantizaciones, devolviendo `501 Not Implemented`).

**Solución:** confirma que tu modelo local soporte solicitudes `OutputSchema` antes de asumir que el código de este módulo es el problema — prueba la misma solicitud contra `MODEL_TYPE=gemini` para aislar si es una limitación del modelo local.
