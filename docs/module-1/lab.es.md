# Reto del Laboratorio 1: Tu Primera Interacción 🗺️

## Objetivo

¡Sin código hoy! Este laboratorio es para ubicarte: vas a explorar la documentación oficial de ADK para Go y el repositorio de código, para que sepas exactamente dónde buscar respuestas más adelante.

## Paso 1: Navega la Documentación Oficial 📖

La documentación oficial de ADK para Go es tu recurso #1 — tutoriales, guías, referencias de API, todo.

1. **Abre la documentación:** ve a `https://adk.dev/get-started/`.
2. **Revisa "Get Started":** lee `Installation` y el quickstart de Go en `https://adk.dev/get-started/go/`. Fíjate en el comando `go get google.golang.org/adk/v2` y el requisito de Go 1.25+.
3. **Explora las secciones de Graph y Workflow:** `https://adk.dev/graphs/` y `https://adk.dev/workflows/` — aquí está el corazón de ADK 2.0, mostrando cómo se combinan nodos y edges en un workflow.

💡 **Punto clave:** asegúrate de estar en las páginas de **Go** específicamente. Go tiene su propia API equivalente a `Runner` (paquete `runner`), separada del shell de app de más alto nivel `cmd/launcher` que usan los ejemplos del quickstart.

## Paso 2: Descubre el Repositorio Oficial de Código 💻

El SDK de Go es open source, con su propio repo en GitHub — código fuente, tracker de issues, y un montón de ejemplos.

1. **Encuéntralo:** `github.com/google/adk-go`.
2. **Verifica la versión:** busca el tag de release `v2.4.0` o superior.
3. **Explora `examples`:** el ejemplo oficial ejecutable más cercano es `examples/workflow/basic`.
4. **Lee el código:** abre `examples/workflow/basic/main.go`. Trata de conectar lo que ves con lo que revisaste en el Paso 1 — `workflow.NewFunctionNode`, `workflow.Chain`, `workflowagent.New`.

🕵️ **¡Búsqueda del tesoro!**
> Encuentra el ejemplo `examples/workflow/basic` e identifica qué es lo que realmente ejecuta el agente de forma programática. Pista: este ejemplo no llama a `runner` directamente — usa un shell de app de más alto nivel. Mira las últimas líneas de `main()`.

💡 **Punto clave:** los ejemplos oficiales son tu mejor fuente de código funcional para aprender (y copiar para tus propios proyectos 😉).

## Paso 3: Encuentra la Comunidad 🙋

1. **Pestaña de Issues:** en `github.com/google/adk-go`, haz clic en "Issues" — bugs, solicitudes de funciones, y una buena forma de entender el estado actual del proyecto.
2. **Pestaña de Discussions:** si está disponible, es el lugar para preguntas y conversaciones de la comunidad.

## Resumen del Laboratorio 🎉

¡Bien hecho, completaste el laboratorio 1! Ahora sabes:

* Cómo navegar la documentación oficial de ADK para Go.
* Dónde encontrar ejemplos oficiales de Go que sí funcionan, en el repo de GitHub.
* Dónde buscar soporte de la comunidad y novedades del proyecto.

Lo que viene: configurar tu propio entorno de desarrollo local en Go para empezar a construir tu primer agente.

## Preguntas de Autorreflexión 🤔

- ¿Por qué es tan importante tener buena documentación oficial y ejemplos para un framework tan nuevo como el SDK de Go?
- Solo por los nombres de archivo en `examples`, ¿qué capacidades avanzadas crees que podría tener el SDK de Go?
- ¿Cómo pueden los canales de la comunidad como GitHub Issues y Discussions acelerar tu propio aprendizaje?

<hr/>

### ¿Buscas la solución? 🔍

Pista: revisa `examples/workflow/basic/main.go` en `github.com/google/adk-go`, específicamente las últimas líneas de `main()` — ahí está la llamada a `cmd/launcher` que buscas. (¿Quieres correr un agente directo desde tu código en vez de a través de una app armada con launcher? Mira el paquete `runner` — `runner.NewInMemory` + `Run`.)
