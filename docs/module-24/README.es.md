# Módulo 24: Evaluando el Desempeño de un Agente (Go) 🧪📊

## Teoría

### Por Qué "Pasó Una Vez" No Es Prueba de Nada

`assert add(2, 2) == 4` funciona porque `add` es determinística. Un agente respaldado por un LLM no lo es: hazle la misma pregunta dos veces y podrías obtener "El resultado es 15" una vez, "Eso da 15" la siguiente — ambas están bien, pero una comparación de igualdad de strings rechazaría la segunda. Y una respuesta final correcta ni siquiera prueba que el agente razonó bien: una suma fácil como `42 + 118` está perfectamente al alcance de la propia aritmética de un modelo "pensante", así que una respuesta que se ve correcta puede ocurrir *sin que el modelo llame jamás a tu herramienta*. Evaluar un agente necesita revisar más que "¿salió bien el texto?".

### La Trayectoria Importa Tanto Como la Respuesta

La secuencia de llamadas a herramientas que hace un agente — qué herramientas, en qué orden, con qué argumentos — es su **trayectoria**, y muchas veces es lo más importante de revisar. Dos agentes pueden llegar a la misma respuesta final idéntica mientras uno llegó ahí llamando a la herramienta correcta con los números correctos y el otro adivinó. Si un cálculo de varios pasos necesita que el propio resultado de `add` se alimente a `multiply` después, y el agente llama a `multiply` primero, su trayectoria está mal aunque de alguna forma termine produciendo un número que se vea correcto.

### La Respuesta de Go: Escribe el Test que lo Demuestre

No hay un framework de evaluación dedicado al cual recurrir aquí — confirmado directamente en el propio código fuente del SDK fijado, `server/adkrest/internal/routers/eval.go`:

> *"ADK Go has no evaluation implementation. The routes exist so the endpoints the web UI calls are recognised and answered deliberately, with 501 and a readable body... Use adk-python for eval workflows."*
>
> (Traducción: "ADK Go no tiene implementación de evaluación. Las rutas existen para que los endpoints que llama la interfaz web sean reconocidos y respondidos deliberadamente, con 501 y un cuerpo legible... Usa adk-python para flujos de evaluación.")

Cada endpoint REST relacionado con evaluación está registrado pero deliberadamente responde `501 Not Implemented`, y el único que sí responde (`metrics-info`) devuelve una lista vacía — su propio comentario: *"que son ninguna."* Así que el test **es** la herramienta. `internal/agents/calculator/golden_path_eval_test.go` graba un "camino dorado" como un struct simple de Go — una pregunta, las llamadas a herramientas esperadas en orden, y un substring que la respuesta final debe contener:

```go
type expectedToolCall struct {
    tool string
    args map[string]any
}

type goldenPathCase struct {
    name                 string
    question             string
    wantTrajectory       []expectedToolCall
    wantResponseContains string
}
```

Un test basado en tabla corre al agente real a través de `runner.Run()`, recolecta cada evento `FunctionCall` en el orden en que llega, y lo verifica contra la trayectoria esperada campo por campo — nombre de herramienta, luego cada argumento, en secuencia — más una verificación con `strings.Contains` (no `==`) sobre la respuesta final, ya que la redacción exacta no debería importar mientras aparezca el número correcto.

### Una Captura Real y Probada — No Solo Plausible

Un test que no puede fallar ante un error real no prueba nada. El propio test de este módulo se revisó de la forma difícil: las dos llamadas esperadas en su caso de varios pasos (`add` luego `multiply`) se intercambiaron deliberadamente, y el test falló de inmediato con un diagnóstico claro y detallado — nombre de herramienta equivocado y argumentos equivocados en ambas posiciones. Revertir el intercambio restauró un pase limpio. Esa es la vara real que una verificación de trayectoria tiene que superar, confirmada en vivo en vez de asumida.

### Puntos Clave ✅
- Una respuesta final que se ve correcta no prueba que un agente razonó bien — podría no haber llamado a ninguna herramienta.
- La **trayectoria** (qué herramientas, en qué orden, con qué argumentos) muchas veces es más reveladora que la respuesta final por sí sola.
- Go no tiene un framework de evaluación dedicado — confirmado directamente desde el propio código fuente del SDK, no inferido de su ausencia en `go doc`.
- La respuesta idiomática de Go es la misma que Go da en todas partes: escribe un test que realmente ejercite el flujo real de eventos, y prueba que puede fallar antes de confiar en que puede pasar.

<hr/>

> **¿Vienes de Python?** 🐍 El propio curso de Python enseña un framework de evaluación completo y dedicado, construido alrededor de esta misma división trayectoria/calidad-de-respuesta: graba una conversación real a través de la pestaña "Eval" de la Dev UI, guárdala como un **Caso de Evaluación** dentro de un **Conjunto de Evaluación** (un archivo `.evalset.json`), y luego reprodúcela más tarde — después de cualquier cambio de prompt o herramienta — vía la Dev UI, el CLI `adk eval`, o `pytest` en CI, calificada contra métricas como `tool_trajectory_avg_score` (coincidencia exacta/en orden/en cualquier orden de llamadas a herramientas), `response_match_score`/`final_response_match_v2` (solapamiento de n-gramas o equivalencia semántica juzgada por LLM), y `hallucinations_v1`/`safety_v1` (verificaciones de fundamentación y contenido dañino). Una capa adicional, **Simulación de Usuario**, deja que un LLM haga de usuario contra un `ConversationScenario` para poner a prueba a un agente contra conversaciones que ningún caso de prueba fijo podría anticipar. Nada de esto — el formato de archivo, el registro de métricas, el flujo de grabación de la Dev UI, el simulador — tiene equivalente en Go; si necesitas esa herramienta exacta, hoy solo vive en `adk-python`. Lo que da el propio test de este módulo es la misma confianza subyacente (¿hace el agente las cosas correctas, en el orden correcto, y dice algo razonable?), solo que como un test de Go escrito a mano y compilado en vez de uno declarativo y calificado por LLM.
