# Laboratorio 21.5: Desafío de Diseño de Arquitectura MAS (Go) 🏗️

## Objetivo

¡Sin código esta vez! Te vas a poner en los zapatos de un Lead AI Architect: dados tres escenarios de negocio, elige el patrón correcto entre los módulos 16-21, esboza la geometría del grafo, y justifica tu elección. 🎯

---

### Escenario 1: El Pipeline de Revisión Legal ⚖️

Un bufete de abogados necesita un sistema para procesar contratos entrantes.
- **Paso A:** Extraer todas las fechas y nombres.
- **Paso B:** Verificar simultáneamente violaciones de "Privacidad" Y riesgos de "Responsabilidad".
- **Paso C:** Si cualquiera de las verificaciones encuentra un riesgo "Alto", enviarlo a un agente Senior Partner para revisión final.
- **Paso D:** Si no, generar un resumen de "Seguro para Firmar".

**Tarea:** Diseña el grafo. ¿Qué primitiva de Go maneja la parte "Simultánea"? ¿Cuál maneja la decisión de "Riesgo Alto"?

<details>
<summary>Piénsalo bien, y luego revisa el razonamiento</summary>

Este es un **híbrido** de dos patrones que ya construimos en este repo:

- El "simultáneamente" del Paso B es un fan-out/fan-in — dos edges compartiendo el mismo `From`, convergiendo en un `workflow.NewJoinNode` antes de que pueda correr el Paso C. Es exactamente la forma propia de `internal/agents/newsaggregator` (Módulo 16): `{From: extractNode, To: privacyNode}`, `{From: extractNode, To: liabilityNode}`, ambos alimentando a un `syncer := workflow.NewJoinNode(...)`.
- El "si cualquiera de las verificaciones es de Riesgo Alto" del Paso C es una decisión de enrutamiento determinista sobre datos que ya se calcularon — sin lógica abierta, sin loop. Eso es Enrutamiento Estructurado del Módulo 17: un pequeño `workflow.NewFunctionNode` que lee los resultados de ambas verificaciones y devuelve un `*session.Event` con `.Routes` puesto en `"senior_partner"` o `"safe_to_sign"`, conectado vía `workflow.EdgeBuilder.AddRoutes`.

Aquí no hace falta ni Orquestación Dinámica (18) ni un loop (20) — todo el camino, incluyendo la ramificación, se conoce de antemano antes de que el grafo corra siquiera. 👍
</details>

---

### Escenario 2: El Escritor de Historias de Múltiples Turnos ✍️

Una agencia creativa quiere un agente que escriba libros infantiles.
- El agente debe escribir un capítulo, y enviarlo a un agente "Crítico".
- Si el Crítico dice "Demasiado Aterrador", el agente debe reescribir el capítulo y enviarlo de vuelta al Crítico.
- Esto continúa hasta que el Crítico esté satisfecho.

**Tarea:** ¿Qué tipo de geometría de grafo es esta? ¿Qué módulo cubrió este patrón?

<details>
<summary>Piénsalo bien, y luego revisa el razonamiento</summary>

Este es el **Workflow Cíclico del Módulo 20** — la forma exacta propia de `internal/agents/essayrefiner`. El número de iteraciones de reescritura no se conoce de antemano, así que no se puede expresar como edges fijos; necesita un `for` real de Go (con tope en `maxIterations`, un límite de seguridad obligatorio) dentro de un `workflow.NewDynamicNode`, llamando a `workflow.RunNode` sobre el crítico y el refinador alternadamente, saliendo temprano en cuanto el crítico aprueba.

Un detalle realmente no obvio que el propio Módulo 20 de este repo confirmó en vivo: el resultado final real del loop no aparece como "el último mensaje de chat que llegó" — llega en un evento terminal distinto, cuyo autor es el nombre del propio workflow agent *raíz*. Si estuvieras construyendo tests automatizados o herramientas de trazas para este escenario exacto, ese es el evento que necesitarías leer.
</details>

---

### Escenario 3: El Bot de Soporte de una Empresa Global 🌍

Una corporación multinacional tiene un agente principal en su sitio web.
- Cuando un usuario pregunta sobre "Envíos en Europa", el agente principal debe hablar con un agente especializado de "Logística de la UE".
- El agente de "Logística de la UE" lo gestiona un equipo distinto en un país distinto, y corre en su propio proyecto seguro de Google Cloud.

**Tarea:** ¿Qué patrón permite que agentes en distintos proyectos/equipos trabajen juntos?

<details>
<summary>Piénsalo bien, y luego revisa el razonamiento</summary>

**Grafos Distribuidos del Módulo 21** — la forma exacta propia de `internal/agents/researchspecialist`/`internal/agents/a2aorchestrator`. El agente de Logística de la UE no es una entrada local de `SubAgents`; es un servicio genuinamente separado, alcanzable vía `agent/remoteagent/v2.NewA2A` y un `remoteagentv2.NewAgentCardProvider` apuntando a su propio `/.well-known/agent-card.json`. El agente principal registra el proxy resultante exactamente igual que un sub-agente local — el mecanismo de delegación (`transfer_to_agent`) es idéntico; solo el transporte cruza una frontera de red real.

Vale la pena recordar de la propia revisión del Módulo 21: el éxito de una llamada remota no se puede probar solo verificando qué nombre de agente aparece como autor de un evento — todos los caminos de fallo marcan ese mismo nombre del wrapper local en su propio evento de error sintetizado. La prueba real está en el contenido: sin errores y no vacío. 🔍
</details>

---

### Preguntas de Autorreflexión 🤔
- ¿Por qué un enfoque "híbrido" (combinando nodos estáticos y dinámicos) suele ser la realidad de los sistemas de producción — y cuál de los seis paquetes propios ya entregados en este repo está más cerca de ser ya un híbrido?
- ¿Cuáles son los riesgos de usar el patrón fluido de "equipo colaborativo" guiado por LLM de `Mode: ModeTask` (Módulo 19) para un proceso financiero estrictamente regulado, comparado con los edges totalmente deterministas de los Módulos 16/17?
- ¿Cómo te ayuda el modelo mental de grafo — nodos y edges que puedes señalar en código Go real — a comunicar el diseño de un sistema a un stakeholder que no es ingeniero, comparado con describirlo como "un chatbot"?

<hr/>

> **¿Vienes de Python?** 🐍 El propio hito de Python plantea estos mismos tres escenarios en términos de `Workflow`, `@node`, `sub_agents`, y `RemoteA2aAgent`. Las respuestas esperadas son idénticas — este laboratorio solo nombra el paquete y la primitiva reales y entregados en Go para cada uno, ya que todos existen como código probado en este repo en vez de un diseño hipotético.
