# Módulo 25.5: IA Responsable (RAI) y Plugins de Seguridad (Go) 🛡️🚫

## Teoría

### Fail-Closed: Revisar Antes de que Llegue al Usuario

El Sistema de Plugins del Módulo 25 *observaba* el comportamiento de un agente. Este módulo lo *controla*. El patrón **Fail-Closed** significa que cada respuesta pasa por una verificación de seguridad obligatoria antes de que el usuario la vea — intercepta la respuesta final, evalúala contra una política, y si viola esa política, bloquéala y sustitúyela por una respuesta segura. No puedes depender solo de los propios filtros internos de un modelo en un entorno empresarial — necesitas una capa programática que haga cumplir tus propias reglas sin importar lo que el modelo mismo decida decir.

### El Mismo Hook, Usado para Reescribir en Vez de Solo Observar

El propio `alertTracker.onEvent` del Módulo 25 usaba `OnEventCallback` para *observar* eventos de respuesta final. Este módulo usa el mismo hook idéntico para *cambiar* uno:

```go
func (g *piiGuardrail) onEvent(_ agent.InvocationContext, event *session.Event) (*session.Event, error) {
	if !event.IsFinalResponse() || event.Content == nil {
		return event, nil
	}
	for _, part := range event.Content.Parts {
		if part.Thought || part.Text == "" || !g.pattern.MatchString(part.Text) {
			continue
		}
		part.Text = safetyMessage
		// ...
	}
	return event, nil
}
```

Modificar `part.Text` en el lugar y devolver el mismo puntero de evento realmente cambia lo que ve el usuario — confirmado leyendo la propia función `fromPlugin` de `runner.go`, no asumido: su propio comentario de documentación dice claramente, *"a plugin that mutates the event in place and returns nil, or returns the same pointer, is the ordinary way to write this hook."* (un plugin que modifica el evento en el lugar y devuelve nil, o devuelve el mismo puntero, es la forma ordinaria de escribir este hook). La modificación llega a cada camino de código que entrega eventos de vuelta al llamador.

### Un Bug Real que el Propio Build de este Módulo Detectó: No Confíes en `Parts[0]`

Construir el propio test en vivo de este módulo sacó a la luz algo que este repositorio ha documentado desde el Módulo 3, pero que nunca había roto código de verdad hasta ahora: el modelo Ollama por defecto es capaz de "pensar", y su respuesta genuinamente se divide en múltiples `Parts` — una parte anterior lleva el rastro de razonamiento (`Thought: true`), y la respuesta real, visible para el usuario, cae en una parte *posterior*, sin marca de pensamiento. La primera versión de este guardrail solo revisaba `Parts[0].Text` y bloqueaba el rastro de razonamiento mientras dejaba pasar directamente el número de tarjeta filtrado de verdad en `Parts[1]`, completamente inadvertido hasta que la propia aserción del test en vivo lo detectó. La solución: revisar cada parte, saltarse las marcadas como `Thought`, y solo inspeccionar (y, si hace falta, reemplazar) el texto genuinamente visible para el usuario.

### Por Qué un Plugin, No una Instrucción

Podrías simplemente decirle al modelo "nunca reveles números de tarjeta de crédito" — pero eso no es cumplimiento forzado, es una petición que el modelo puede ignorar, malinterpretar, o esquivar con el prompt correcto. Un plugin es código: corre en cada respuesta individual, para cada agente en el que esté registrado, sin importar lo que diga la propia instrucción de ningún agente en particular. Por eso también pertenece al nivel del Runner/launcher (`PluginConfig`), la misma ubicación que estableció el módulo 25 — un solo plugin de guardarraíl puede proteger a cada agente de una organización sin tocar ninguno de sus propios prompts.

### Puntos Clave ✅
- **Fail-Closed** significa interceptar y evaluar una respuesta *antes* de que el usuario la vea, no filtrarla después del hecho.
- `OnEventCallback` puede reescribir un evento, no solo observarlo — modifica el `Text` de una parte en el lugar y devuelve el mismo evento, confirmado que realmente cambia la respuesta visible para el usuario.
- La respuesta de un modelo pensante son varias `Parts`; una verificación de seguridad que solo inspecciona `Parts[0]` puede perderse la respuesta real por completo — revisado en vivo, no asumido.
- Un plugin de seguridad, registrado una sola vez a nivel del Runner/launcher, protege a cada agente al que ese plugin esté conectado — sin ninguna instrucción por agente que mantener ni en la que confiar.

<hr/>

> **¿Vienes de Python?** 🐍 El propio módulo de Python cubre el mismo patrón Fail-Closed y la misma intercepción con `on_event_callback` (`BasePlugin`, `event.content.parts[0].text = "..."`). El mecanismo coincide directamente — lo único que vale la pena señalar: el propio ejemplo de Python también revisa solo `parts[0]`, lo cual funciona bien contra la salida plana (sin pensamiento) de Gemini, pero necesitaría la misma solución de "revisar cada parte que no sea de pensamiento" que hizo este módulo si alguna vez corriera contra un modelo capaz de pensar.
