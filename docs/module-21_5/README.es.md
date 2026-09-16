# Módulo 21.5: Hito de Conocimiento de MAS — Elección de Arquitectura (Go) 🏆

## Teoría

Ya construiste seis sistemas multi-agente reales y funcionando en este repo, cada uno resolviendo la orquestación de una forma genuinamente distinta. ¡Bien hecho! Elegir la geometría correcta para un grafo nuevo — *antes* de escribir código — es la decisión más importante que enseña toda esta sección del curso.

### Los Seis Patrones, Repasados 📋

| Patrón | Módulo | Paquete Go (entregado, en este repo) | Primitivas Clave de Go |
|---|---|---|---|
| Orquestación Estática | [16](../module-16/README.md) | `internal/agents/newsaggregator` | `workflow.Edge`, `workflow.NewJoinNode` |
| Enrutamiento Estructurado | [17](../module-17/README.md) | `internal/agents/marketrouter` | `workflow.NewFunctionNode` devolviendo `*session.Event` con `.Routes` establecido, `workflow.EdgeBuilder.AddRoutes` |
| Orquestación Dinámica | [18](../module-18/README.md) | `internal/agents/supportrouter` | `workflow.NewDynamicNode`, `workflow.RunNode` |
| Equipos Colaborativos | [19](../module-19/README.md) | `internal/agents/travelplanner` | `llmagent.Config.Mode` (`ModeTask`/`ModeSingleTurn`) |
| Workflows Cíclicos | [20](../module-20/README.md) | `internal/agents/essayrefiner` | un `for` normal de Go dentro de `workflow.NewDynamicNode`, llamando a `workflow.RunNode` repetidamente |
| Grafos Distribuidos | [21](../module-21/README.md) | `internal/agents/researchspecialist`, `internal/agents/a2aorchestrator` | `cmd/launcher/web/a2a`, `agent/remoteagent/v2.NewA2A` |

Cada link va directo al propio README de ese módulo — el trabajo de este hito es el repaso y el marco de decisión, no una segunda copia de la Teoría de seis módulos.

### Cómo Elegir, en Términos Propios de Go 🧭

Tres preguntas, traducidas de lo abstracto a lo que realmente significan cuando tienes código Go real en las manos:

1. **¿El camino es predecible?** Si cada rama posible se puede declarar de antemano como un `workflow.Edge` — secuencial, paralela, o etiquetada con `Route` — ve por **Estático** (16) o **Enrutamiento Estructurado** (17). Son los más baratos de razonar: la lista de edges *es* todo el grafo.
2. **¿La lógica necesita flujo de control normal de Go — un `for`, una cadena `if`/`else`, una rama estilo `try`/`recover` — que no se reduce a un conjunto fijo de edges?** Usa un nodo **Dinámico** (18) — un `workflow.NewDynamicNode` cuyo cuerpo es solo código Go llamando a `workflow.RunNode`. Si esa lógica necesita *repetirse* hasta que se cumpla alguna condición decidida por el modelo, eso es específicamente la forma **Cíclica** (20) — el mismo mecanismo `NewDynamicNode`/`RunNode`, con un loop y un tope duro de iteraciones.
3. **¿Los agentes necesitan vivir en entornos genuinamente separados** — distintos procesos, distintas máquinas, distintos repositorios de distintos equipos? Eso es **Distribuido** (21) — `agent/remoteagent/v2.NewA2A` sobre A2A HTTP real, no una entrada local de `SubAgents`.

Entonces, ¿dónde encajan los **Equipos Colaborativos** (19)? Es el que sobra a propósito 😄 — no se trata para nada de la *forma* del grafo, sino de cuánto control le cedes al propio sub-agente. `Mode: ModeSingleTurn`/`ModeTask` te da delegación guiada por el LLM con un retorno *garantizado*, sin escribir código de orquestación — la elección correcta cuando el propio juicio del especialista sobre cuándo está "listo" es suficiente, y prefieres no escribir tú mismo el loop o el edge de enrutamiento.

### Detalles que Vale la Pena Recordar 🔍

Dos de los seis módulos propios de este repo confirmaron algo que realmente vale la pena recordar, mucho más allá de "cómo construir el patrón":

- **El Módulo 19** confirmó que `llmagent.Config` no necesita *ninguna* configuración de reanudación para la pausa/reanudación multi-turno del modo tarea — el framework configura el equivalente a nivel de workflow automáticamente para cada nodo `LlmAgent`.
- **El Módulo 20** confirmó que el resultado propio del loop de un nodo dinámico vive en un evento terminal distinto, cuyo autor es el nombre del workflow agent *raíz*, con `Content` en nil y `Output` establecido — no en el evento de contenido de chat que resulte quedar de último.

Ambos son justo el tipo de detalle que decide si tu propio test (o tu propio código de inspección de trazas) realmente prueba lo que crees que prueba. La revisión del propio Módulo 21 detectó exactamente esta categoría de error en un *test*, no solo en documentación: una verificación que solo miraba el autor de un evento, que resultó no decir nada sobre si la llamada realmente había tenido éxito.

### Puntos Clave ✅
- No existe una arquitectura única para todo — la mayoría de las construcciones más grandes y posteriores de este propio repo combinan estos patrones (piensa en un fan-out estático que alimenta un loop dinámico).
- Prefiere la geometría más simple que resuelva el problema: no vayas por un `NewDynamicNode` si una lista fija de `workflow.Edge` ya dice todo lo que necesitas.
- Cada patrón que repasa este hito tiene un paquete Go real y probado en vivo, justo aquí en este repo — cuando tengas dudas sobre el comportamiento exacto de un mecanismo, los propios tests y README de ese paquete son la verdad, no esta tabla de repaso.

<hr/>

> **¿Vienes de Python?** 🐍 El propio hito de Python repasa `Workflow`/`@node`/`ctx.run_node()`/`sub_agents`/`RemoteA2aAgent` por nombre. Este espejo en Go repasa los mismos seis patrones apuntando a paquetes Go reales y ya entregados en su lugar — cada uno ya verificado de forma independiente en vivo durante su propio módulo, incluyendo al menos dos lugares (los módulos 19 y 20) donde el mecanismo de Go resultó comportarse de una forma genuinamente mejor o más sorprendente de lo que describe el propio material de Python, no solo un equivalente renombrado.
