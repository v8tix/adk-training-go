# Solución de Problemas: Módulo 18 (Go) 🛠️

### `workflow.RunNode` falla con "output type ... does not satisfy expected ..." ❌

**Síntoma:** llamar a `workflow.RunNode[SomeStruct](ctx, classifierNode, input)` contra un nodo construido con `llmagent.Config.OutputSchema` falla en tiempo de ejecución con un error como `workflow.RunNode: child "classifier" output type map[string]interface {} does not satisfy expected main.SomeStruct`.

**Causa:** confirmado en vivo este módulo — la implementación de `workflow.RunNode` hace una simple aserción de tipo de Go sobre la salida cruda del nodo hijo, sin conversión JSON consciente de schema (a diferencia de la coerción de input de `workflow.NewFunctionNode`, que sí convierte). El resultado estructurado de un agente restringido por `OutputSchema` siempre llega a `RunNode` como `map[string]any`, nunca como un struct tipado.

**Solución:** declara la llamada como `workflow.RunNode[map[string]any](ctx, child, input)` e indexa el resultado por key (p. ej. `result["sentiment"]`), en vez de declarar un struct de Go que haga match como `OUT`.

### El clasificador clasifica mal un mensaje límite 🤔

**Causa:** confirmado en vivo este módulo — un mensaje genuinamente ambiguo ("My internet is down, help!") fue clasificado como "angry" tanto por Ollama como por Gemini en el propio probe de este curso, cuando un lector humano podría llamarlo neutral. Esta es una limitación de calidad del modelo y del prompt del clasificador, no un bug en el código de enrutamiento — el `if`/`else` siguió correctamente lo que sea que el clasificador realmente devolvió.

**Solución:** usa mensajes de test inequívocos (claramente enojados, claramente felices) al verificar la lógica de enrutamiento en sí. Si la precisión de clasificación en mensajes límite importa para tu propio caso de uso, eso es un problema de ingeniería de prompts o elección de modelo a resolver por separado del patrón de orquestación que enseña este módulo.
