# Laboratorio 14: Integrando una Herramienta de Wikipedia de Terceros (Go) 📚

## Objetivo

Construye un agente "fact-finder" que busca información en Wikipedia, usando un paquete Go de terceros real e independientemente mantenido — sin necesitar cuenta cloud más allá de lo que ya piden módulos anteriores.

### Paso 1: La Herramienta

`internal/agents/factfinder/tools.go`'s `lookupWikipedia` llama a la función `Summary` de `github.com/trietmn/go-wiki`:

```go
func lookupWikipedia(_ agent.Context, args LookupWikipediaArgs) (LookupWikipediaResult, error) {
    summary, err := gowiki.Summary(args.Query, 5, -1, false, true)
    if err != nil {
        return LookupWikipediaResult{Status: "error", Error: err.Error()}, nil
    }
    return LookupWikipediaResult{Status: "success", Summary: summary}, nil
}
```

Fíjate en el `init()` propio del paquete, justo arriba — ahí vive el arreglo del User-Agent del README de este módulo.

### Paso 2: El Agente

`internal/agents/factfinder/agent.go`'s `BuildRootAgent` lo envuelve vía `functiontool.New`, exactamente igual que `calculator.BuildRootAgent` del módulo 9:

```go
wikipediaTool, err := functiontool.New(functiontool.Config{
    Name:        "lookup_wikipedia",
    Description: "Looks up a topic on Wikipedia and returns a short summary.",
}, lookupWikipedia)
```

### Paso 3: Córrelo 🚀

```bash
go run ./cmd/fact-finder console
```

Salida real y confirmada de este comando exacto (el texto de razonamiento del modelo local corre directo hacia su respuesta sin separador — `cmd/launcher/console` imprime cada parte de texto sin condiciones, sin filtrar las partes `Thought`, un detalle conocido del renderizado del SDK cubierto en módulos anteriores, no algo que el código de este módulo pueda arreglar):

```
📚 fact-finder using qwen3.8:27b

User -> Who was Marie Curie?
Agent -> The user is asking a factual, encyclopedia-style question about Marie Curie.
According to my instructions, I should use the lookup_wikipedia tool to answer this
rather than from memory alone.The tool returned a successful result with a summary
about Marie Curie. I can now use this information to answer the user's question.
**Marie Curie** (born Maria Salomea Skłodowska; 7 November 1867 – 4 July 1934) was a
Polish and naturalised-French physicist and chemist. Here are some highlights of her
life and achievements:

- **Nobel Prizes:** She won the 1903 Nobel Prize in Physics (shared with her husband,
  Pierre Curie) and the 1911 Nobel Prize in Chemistry.
- **Records:** She was the first woman to win a Nobel Prize, the first person to win
  two, and the only person to win in two different scientific fields.
```

No hace falta `GOOGLE_AI_STUDIO_API_KEY` — esto corre por completo en el Ollama local por defecto, confirmado en vivo, ya que una herramienta de función simple no necesita capacidad de herramienta integrada. ¿Quieres ver que la herramienta realmente verificó que buscó en vez de adivinar? Maneja el agente directamente vía `runner.Run` en vez del launcher de consola, como hace `askFactFinder` en `agent_test.go` — filtra las partes `Thought` y lee el propio `FunctionResponse` de la herramienta, el mismo patrón que este repo usa para probar cada agente desde el módulo 9.

### Paso 4: Confirma que la Herramienta Realmente Se Ejecutó ✅

`assertLooksUpWikipedia` en `internal/agents/factfinder/agent_test.go` revisa el propio `FunctionResponse` de la herramienta — no solo el texto final — verificando `status == "success"` y que el resumen realmente mencione el tema consultado. Esto importa porque un modelo podría responder una pregunta tan conocida como esta directo desde sus propios datos de entrenamiento, sin llegar a llamar a la herramienta jamás. 🕵️

### Troubleshooting

Ve [troubleshooting.md](./troubleshooting.md) si algún paso no se comporta como esperabas.

### Resumen del Laboratorio 🎉

Integraste un paquete Go de terceros real en un agente ADK usando exactamente el mismo patrón `functiontool.New` que todo módulo de herramientas personalizadas usa desde el módulo 9 — sin necesitar ningún adaptador especial. ¡Fácil!

### Preguntas de Autorreflexión 🤔
- ¿Por qué `functiontool.New` no necesita saber de dónde viene la lógica de `lookupWikipedia` — tu propio código o un paquete de terceros?
- La prueba de "consulta sin sentido" de `TestLookupWikipedia` verifica un resultado de error estructurado, no un error de Go. ¿Por qué importa esa distinción para lo que el modelo puede hacer con el resultado?
- ¿Qué cambiaría en `tools.go` si quisieras cambiar `github.com/trietmn/go-wiki` por una librería cliente de Wikipedia distinta?

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python te hace instanciar un `WikipediaAPIWrapper`, envolverlo en `WikipediaQueryRun`, y luego envolver *eso* en `LangchainTool` — tres capas, porque el objeto herramienta de LangChain necesita traducirse a la forma del ADK. Este laboratorio tiene una sola capa: una función Go que llama directamente al paquete de terceros, envuelta con el mismo `functiontool.New` que ya usa todo módulo anterior.
