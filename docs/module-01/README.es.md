# Módulo 1: Introducción a los Agentes de IA y Google ADK (Go) 🤖

## Teoría

### Por qué los Agentes de IA son la nueva ola 🚀

¿Chatbots que solo responden preguntas? Ya quedó atrás. Lo interesante ahora son los **Agentes de IA**: sistemas que entienden un objetivo, arman un plan, y realmente *hacen* cosas para cumplirlo.

La diferencia clave con un programa normal: un script sigue instrucciones fijas — si pasa X, hago Y. Un agente razona, se adapta y actúa por su cuenta. Los LLMs como Gemini de Google son los que hacen esto posible — son el "cerebro" detrás de todo.

### Entonces, ¿qué es exactamente un Agente? 🧠

Un Agente de IA puede:

1. **Percibir** — recibir información, como una solicitud en lenguaje natural.
2. **Razonar y Planificar** — usar un LLM para dividir un objetivo grande en pasos más pequeños y ejecutables.
3. **Actuar con Herramientas** — ejecutar esos pasos de verdad: llamar una API, buscar en una base de datos, correr código, o incluso delegarle trabajo a otro agente.
4. **Observar y Ajustar** — revisar cómo salió todo y ajustar el plan si hace falta, hasta lograr el objetivo.

Piénsalo así: no es "chatear con una IA", es delegarle una tarea a un colega capaz. 💪

### Te presentamos el Google Agent Development Kit (ADK) — ¡ahora en Go! 🎉

Construir un agente listo para producción es mucho más que darle un prompt a un LLM. Hay que manejar el historial de conversación, integrar herramientas, orquestar flujos complejos, evaluar el rendimiento y desplegar todo en infraestructura escalable.

Justo eso resuelve el **Google Agent Development Kit (ADK)**: te permite construir, gestionar, evaluar y desplegar agentes en la **Gemini Enterprise Agent Platform** (la que quizás conoces como Vertex AI). Y desde ADK 2.0, ya existe un SDK oficial para Go: `google.golang.org/adk/v2` (código fuente: `github.com/google/adk-go`, con el tag actual `v2.4.0`; requiere Go 1.25+ — este curso usa Go 1.27). Instálalo así:

```bash
go get google.golang.org/adk/v2
```

📚 Documentación completa en `https://adk.dev/get-started/go/`.

#### La filosofía del ADK

Tres palabras: **modularidad, flexibilidad, escalabilidad**. Te da un conjunto de piezas básicas que puedes combinar como bloques de LEGO — desde un agente simple de un solo propósito, hasta un sistema completo multi-agente.

#### La idea central de ADK 2.0: todo es un grafo 🕸️

Olvídate de los agentes monolíticos y las jerarquías rígidas — ADK 2.0 piensa en **grafos**. Así se traduce cada concepto al SDK de Go:

| Concepto de arquitectura de grafo | Go ADK (`google.golang.org/adk/v2`) |
|---|---|
| **Node** (nodo) — una unidad de trabajo | `workflow.NewFunctionNode(name, fn, workflow.NodeConfig{...})` para código plano, o un agente `llmagent.New(...)` usado directamente como nodo para pasos con LLM |
| **Edge** (arista) — el flujo de control y datos entre nodos | `workflow.Chain(workflow.Start, nodeA, nodeB, ...)` para flujo secuencial; valores explícitos `workflow.Edge{}` con `StringRoute`/`IntRoute`/`BoolRoute` para ramificaciones |
| **Workflow** — el contenedor y orquestador | `workflowagent.New(workflowagent.Config{Name, Description, Edges: edges})` — envuelve un conjunto de edges para que cumpla la misma interfaz que un agente individual |
| **App & Runner** — la capa de infraestructura | `runner.NewInMemory(appName, agent)` construye un runner que manejas directamente desde tu propio código: `(*Runner).Run(ctx, userID, sessionID, msg, cfg, opts...)` va entregando los eventos del agente uno por uno. El quickstart de "get-started" en cambio usa el shell de más alto nivel `cmd/launcher` (`full.NewLauncher()` + `agent.NewSingleLoader(...)`) para obtener gratis una app completa de CLI/dev-UI/API-server. Usa `runner` directamente cuando solo necesitas correr un agente desde un script; usa `cmd/launcher` cuando quieres un shell de app ya armado. También existe un tipo `App` para casos avanzados (plugins, caché, ciclo de vida). |
| **Tool** (herramienta) — interfaces de capacidades | Interfaz `tool.Tool`; ya vienen integradas como `tool/geminitool.GoogleSearch{}`, y puedes crear las tuyas con `tool/functiontool` |
| **Session & State** — contexto y memoria durante una ejecución | Paquete `session` |

A lo largo de este curso vas a aprender a pensar en **Grafos y Nodos**, dominando ADK 2.0 para Go y construyendo aplicaciones de IA escalables y listas para producción. ¡Vamos! 🚀

### Puntos clave ✅

- Los Agentes de IA perciben, razonan y actúan usando herramientas — de forma autónoma.
- **ADK 2.0** piensa en una **Arquitectura de Grafo**: agentes y herramientas son **Nodos**, conectados por **Edges** — y sí, ya está disponible para Go vía `google.golang.org/adk/v2`.
- Los agentes construidos con ADK se despliegan todos en la **Gemini Enterprise Agent Platform** (antes Vertex AI), sin importar con qué SDK de lenguaje los construiste.
- Go te da `runner.NewInMemory` + `Run` para manejar un agente directo desde tu código, y `cmd/launcher` cuando quieres una app de CLI/dev-UI ya lista — elige el que mejor encaje con tu proyecto.

<hr/>

> **¿Vienes de Python?** 🐍 El `runner.NewInMemory` de Go es la misma idea que el `InMemoryRunner` de Python — su propio comentario de documentación lo dice directamente — y `cmd/launcher` es lo más parecido al shell de app ya armado de `adk web`.
