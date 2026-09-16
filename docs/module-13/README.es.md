# Módulo 13: Interacciones Avanzadas — Actions y HITL (Go) 🛑

## Teoría

### Una Herramienta que se Pausa para un Humano 🧍

Cada herramienta personalizada hasta ahora se ejecutaba por completo apenas el modelo la llamaba. Pero algunas acciones — mover dinero, borrar datos, enviar un mensaje — realmente no deberían dispararse solo porque un modelo *decidió* hacerlo. `functiontool.Config` tiene un campo hecho justo para esto:

```go
investmentTool, err := functiontool.New(functiontool.Config{
    Name:                "execute_investment",
    Description:         "Executes a long-term investment.",
    RequireConfirmation: true,
}, executeInvestment)
```

Pon `RequireConfirmation: true`, y el framework intercepta la primera llamada — `executeInvestment` todavía no se ejecuta. En cambio, emite un evento especial (un `FunctionCall` llamado `adk_request_confirmation`) y le dice al modelo "esto necesita confirmación." Confirmado en vivo: la primera línea de tu handler real nunca se ejecuta hasta que un humano de verdad responda. 🔒

### Respondiendo la Confirmación ✅❌

La aplicación que hace la llamada responde ese evento especial con un `FunctionResponse` del mismo nombre, con `{"confirmed": true}` o `{"confirmed": false}`:

```go
confirmResp := &genai.Content{
    Role: string(genai.RoleUser),
    Parts: []*genai.Part{{
        FunctionResponse: &genai.FunctionResponse{
            Name:     toolconfirmation.FunctionCallName, // "adk_request_confirmation"
            ID:       confirmCallID,                     // el ID de la propia llamada de confirmación
            Response: map[string]any{"confirmed": true},
        },
    }},
}
```

Envía eso en el siguiente turno y el framework retoma la llamada *original* a `execute_investment` — tu handler real se ejecuta ahora, con los mismos argumentos que el modelo dio la primera vez. Recházala (`"confirmed": false`), y el handler nunca se ejecuta — la llamada falla con `tool.ErrConfirmationRejected`. Confirmado en vivo: el rechazo lo impone el framework mismo, no algo que tu código de handler tenga que revisar. 👍

¿Estás armando una CLI interactiva? Esto te lo regalan: `cmd/launcher/console` ya sabe reconocer los eventos `adk_request_confirmation` y pregunta sí/no automáticamente (confirmado leyendo su propio código fuente `hitl.go`). `cmd/finance-agent` en este módulo usa el launcher plano, igual que todos los programas `cmd/` anteriores, y simplemente... funciona. Sin código extra. ✨

### Controlando el Runtime con `ctx.Actions()` 🎛️

`agent.Context.Actions()` te da el `*session.EventActions` del evento actual — una forma de que una herramienta influya en lo que pasa después, más allá de su propio valor de retorno:

```go
func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
    if args.Amount > escalationThreshold {
        ctx.Actions().TransferToAgent = "supervisor"
        return ExecuteInvestmentResult{Status: "escalated"}, nil
    }
    return ExecuteInvestmentResult{Status: "success"}, nil
}
```

`SkipSummarization` (usado internamente por el mecanismo de confirmación de arriba) le dice al framework "no dejes que el modelo reescriba este resultado antes de mostrárselo al usuario." `TransferToAgent` le pasa toda la conversación a otro agente — `supervisor` aquí no necesita ningún wrapper especial, es un `llmagent` plano, conectado vía el propio `SubAgents: []agent.Agent{supervisorAgent}` del agente financiero. No hace falta el paquete `workflow`/`Workflow` — confirmado en vivo, un agente plano con sub-agentes hace el trabajo solo. 🙌

### Un Detalle Real que Confirmamos: Confirmación + Transferencia en la Misma Llamada 🔍

Si pones `ctx.Actions().TransferToAgent` desde una herramienta plana e incondicional (sin puerta de confirmación), el agente activo cambia de inmediato — el siguiente evento viene del nuevo agente, sin que el modelo decida nada. Pero `executeInvestment` de arriba *también* está envuelto con `RequireConfirmation: true`, y esa combinación se comporta distinto. Confirmado en vivo: la transferencia **no** pasa en el mismo turno en que se resuelve la llamada confirmada. En cambio, el modelo recibe un turno más dentro del agente original, ve el estado "escalated," y tiene que llamar él mismo a la herramienta `transfer_to_agent` que el framework inyecta automáticamente — *solo entonces* cambia el agente activo de verdad. Tanto Ollama como Gemini lo lograron de forma confiable una vez instruidos, pero hizo falta una línea explícita en el prompt para que fuera confiable (mira las Constraints más abajo) — sin ella, un modelo "pensante" puede simplemente narrar la escalación en texto en vez de llamar realmente a la herramienta. ¡Tramposo! 😅

Y otro hallazgo real de construir este módulo: una confirmación rechazada, si la instrucción del agente no deja claro qué pasa después, puede meter al modelo en un ciclo raro — rechaza → escala → el supervisor lo rebota → reintenta → rechaza otra vez, y así. `internal/agents/financeagent/prompts/finance_instruction.md` le dice explícitamente al modelo que un rechazo es definitivo, punto. Esa línea no era opcional — una instrucción genérica sola dejaba al modelo improvisar, y lo hacía mal.

### Leyendo Archivos Subidos: `ctx.Artifacts().Load` 📎

La tercera capacidad de `ToolContext` de la teoría de este módulo — leer un archivo que subió el usuario — tiene una construcción directa, aunque este módulo no la construye ni la prueba (solo de referencia):

```go
func analyzeLogs(ctx agent.Context, args AnalyzeLogsArgs) (AnalyzeLogsResult, error) {
    resp, err := ctx.Artifacts().Load(ctx, args.FileName)
    if err != nil {
        return AnalyzeLogsResult{}, err
    }
    // resp holds the artifact's content as a *genai.Part.
}
```

### Puntos Clave ✅
- `functiontool.Config{RequireConfirmation: true}` pausa una herramienta para aprobación humana — el handler real nunca corre hasta que un `FunctionResponse` llamado `toolconfirmation.FunctionCallName` lo confirma, y el rechazo lo impone el framework, no tu código.
- `cmd/launcher/console` ya habla este protocolo de confirmación — una CLI interactiva no necesita código adicional para eso.
- `ctx.Actions().TransferToAgent` le pasa la conversación a una entrada de `SubAgents` — no hace falta un wrapper `Workflow`.
- Combinar confirmación y transferencia en la misma llamada de herramienta necesita un turno extra del modelo antes de que la transferencia realmente ocurra — una secuencia real y confirmada, no una suposición.
- `ctx.Artifacts().Load` es el equivalente directo para leer archivos subidos — una construcción real aunque este laboratorio no la implemente.

<hr/>

> **¿Vienes de Python?** 🐍 `functiontool.Config.RequireConfirmation` cumple el mismo rol que `FunctionTool(fn, require_confirmation=True)` en Python; `ctx.Actions()` es el mismo objeto que `tool_context.actions`, con los mismos campos `transfer_to_agent`/`skip_summarization`; `ctx.Artifacts().Load` es `tool_context.load_artifact`. El propio README de Python presenta su contenedor `Workflow` como una elección de estilo, no un requisito, para este laboratorio — confirmado también válido en Go: un `llmagent` plano con `SubAgents` es todo lo que necesitas.
