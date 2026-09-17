# Laboratorio 23: Construyendo un Pipeline de Procesamiento de Documentos (Go) 📄🖼️

## Objetivo

Construye un agente **Procesador de Documentos** que corra un pipeline de cuatro pasos — extraer, resumir, graficar, reportar — guardando la salida de cada paso como un artifact versionado, encadenando la salida de cada paso hacia el siguiente.

## Tareas del Laboratorio

### 1. Lee `internal/agents/documentprocessor/tools.go`

Cuatro handlers, cada uno mostrando una parte distinta del sistema de Artifacts:

- `extractText` — guarda texto limpio como `"{document_name}_extracted.txt"` vía `ctx.Artifacts().Save`. Llámalo dos veces sobre el mismo documento y obtienes las versiones 1, luego 2 — nunca una sobrescritura.
- `summarizeDocument` — primero carga el artifact de texto extraído. Si todavía no existe (`errors.Is(err, fs.ErrNotExist)`), devuelve un resultado normal y exitoso con un mensaje útil en vez de un error de Go — el modelo puede reaccionar a eso en la conversación en vez de fallar.
- `generateChart` — el único artifact binario de este módulo: un PNG de prueba hardcodeado, guardado vía `genai.NewPartFromBytes(pngBytes, "image/png")`.
- `createReport` — lista todos los artifacts (`ctx.Artifacts().List`), filtra a los archivos propios de este documento, y decide según `part.InlineData.MIMEType` para distinguir la imagen del gráfico de los artifacts de texto al compilar el reporte final.

### 2. Lee `internal/agents/documentprocessor/agent.go` y su prompt

Misma forma con `functiontool.New` + `Tools` que el agente de todo módulo anterior. Su instrucción (`prompts/document_processor_instruction.md`) le dice al modelo que corra los cuatro pasos estrictamente en orden, ya que cada uno lee lo que el anterior guardó.

### 3. Córrelo — modo consola, completamente local 🖥️

```bash
go run ./cmd/document-processor console
```

Sin `.env`, sin API key. Salida real y confirmada de este comando exacto (razonamiento del modelo de pensamiento recortado para legibilidad):

```
📄 document-processor using qwen3.8:27b

User -> Process the document named 'AnnualReport'.
Agent -> All done! Here's a summary of the processing for **AnnualReport**:

1. **Extracted text** → saved as `AnnualReport_extracted.txt` (version 1)
2. **Summary** → saved as `AnnualReport_summary.txt` (version 1)
3. **Stats chart** → saved as `AnnualReport_chart.png` (version 1)
4. **Final report** → saved as `AnnualReport_FINAL_REPORT.md` (version 1)

All four files were generated successfully and are ready for your use. Let me know if you'd like anything else!
```

Un mensaje de usuario, cuatro llamadas a herramientas encadenadas, cada una guardando un artifact real y versionado — confirmado de forma estructural, no solo por esta transcripción, en `internal/agents/documentprocessor/agent_test.go`.

**Nota sobre el cableado de `ArtifactService`:** a diferencia de todo `cmd/` anterior, el `main.go` de este establece `ArtifactService: artifact.InMemoryService()` explícitamente en `launcher.Config`. Eso es requerido, no decorativo — `runner.New` nunca pone un `ArtifactService` nulo por defecto, a diferencia del propio cableado interno del modo `web`; sáltatelo, y cualquier herramienta que llame a `ctx.Artifacts().Save(...)` hace panic sobre una interfaz nula. Si estás extendiendo este laboratorio hacia un nuevo programa `cmd/` propio que toque artifacts, no olvides esta línea.

### 4. Lee `internal/agents/documentprocessor/tools_test.go`

Tests unitarios puros, sin LLM — un nuevo par `fakeArtifacts` (respaldado por un mapa, versiones indexadas desde 1, igualando exactamente a los backends reales) más `erroringArtifacts` para tests de propagación. Cubre el incremento de versiones, la distinción entre no-encontrado y error real, la corrección del MIME type, y — un caso límite real que vale la pena notar — `createReport` excluyendo su propia versión previa cuando se corre dos veces sobre el mismo documento.

### 5. Lee `internal/agents/documentprocessor/artifacts_test.go`

Tres tests independientes contra el `artifact.InMemoryService()` *real*, sin LLM: las versiones genuinamente empiezan en 1, un artifact con prefijo `user:` genuinamente cruza sesiones, y — el caso negativo — un nombre de archivo simple genuinamente no lo hace. Esto es lo que realmente prueba las afirmaciones de alcance y versionado, no solo el doble falso del que dependen los propios tests unitarios del módulo 4.

### 6. Lee `internal/agents/documentprocessor/agent_test.go`

`TestDocumentProcessor_BuildsVersionedPipeline_{Ollama,Gemini}` corre todo el pipeline a través de una sola llamada a `Run()`, y luego lee el almacén de artifacts directamente después: los cuatro archivos existen, el texto extraído es genuinamente la versión 1, el MIME type del gráfico es genuinamente `image/png`, y el propio contenido del reporte final genuinamente referencia el nombre de archivo del gráfico — la prueba real y estructural de que los cuatro pasos se encadenan correctamente.

## Preguntas de Autorreflexión 🤔
- ¿Por qué `summarizeDocument` devuelve un resultado normal (no un error de Go) cuando el artifact de texto extraído falta, mientras que una falla real del servicio sigue propagándose como un error real?
- ¿Qué pasaría si `createReport` no excluyera su propio nombre de archivo de la lista que procesa? Prueba quitando esa verificación y volviendo a correr `TestCreateReport_ExcludesItsOwnPriorVersion` para verlo.
- Si quisieras que un gráfico persistiera a través de cada sesión que un usuario inicie (no solo esta), ¿qué cambio de un solo carácter en su nombre de archivo harías?
- ¿Por qué el modo `console` necesita que `ArtifactService` se establezca explícitamente en `main.go`, mientras que el modo `web` no?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee `internal/agents/documentprocessor/tools.go` y `agent.go` para el mecanismo real — cuatro funciones de herramienta, cada una tocando `ctx.Artifacts()`, envueltas de la misma forma que las herramientas de todo módulo anterior.
