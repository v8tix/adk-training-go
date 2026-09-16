# Laboratorio 13: Construyendo un Agente Financiero Seguro con HITL y Actions (Go) 💰

## Objetivo

Construye un agente financiero que requiera confirmación humana antes de cada inversión, y que escale automáticamente las inversiones grandes a un supervisor — usando `functiontool.Config.RequireConfirmation` y `ctx.Actions().TransferToAgent` del README de este módulo.

### Prerrequisitos

Cualquiera de los dos backends funciona — no hace falta `GOOGLE_AI_STUDIO_API_KEY`, confirmado en vivo en este módulo (tanto Ollama como Gemini soportan de verdad el ida-y-vuelta de confirmación y la transferencia dinámica). 🎉

### Paso 1: La Herramienta de Inversión

`internal/agents/financeagent/tools.go`'s `executeInvestment`:

```go
const escalationThreshold = 10000

func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
    if args.Amount > escalationThreshold {
        ctx.Actions().TransferToAgent = "supervisor"
        return ExecuteInvestmentResult{Status: "escalated"}, nil
    }
    return ExecuteInvestmentResult{Status: "success"}, nil
}
```

Léelo junto al `BuildRootAgent` de `agent.go`, que lo envuelve con `functiontool.Config{RequireConfirmation: true}` y conecta un sub-agente `supervisor` vía `SubAgents`.

### Paso 2: Córrelo de Forma Interactiva 🖥️

```bash
go run ./cmd/finance-agent console
```

Salida real y confirmada de este comando exacto (inversión pequeña, aprobada):

```
💰 finance-agent using qwen3.8:27b

User -> Invest $500 for me.
Agent -> The user explicitly asks to invest $500. I should call execute_investment with amount 500.
Agent -> Confirm execute_investment?
  Type 'yes' to confirm, anything else to reject.
User -> yes
Agent -> Great news! Your investment of $500 was executed successfully.
```

Salida real y confirmada de este comando exacto (inversión grande, aprobada, escalada; abreviada — el razonamiento del supervisor es más largo):

```
💰 finance-agent using qwen3.8:27b

User -> Invest $50000 for me.
Agent -> The user is asking to invest $50,000. I need to call the `execute_investment` tool
with amount 50000. This requires human approval before it runs, so I'll make the call and
wait for the result.
Agent -> Confirm execute_investment?
  Type 'yes' to confirm, anything else to reject.
User -> yes
Agent -> The tool result has status "escalated", so I must transfer the conversation to the
supervisor agent using the transfer_to_agent tool before saying anything else to the user.
[...]

**Verdict: Approved.**

Your request to invest $50,000 is approved. The user has clearly and explicitly authorized
this amount, and the trade is now cleared to proceed. The finance_agent will execute the
investment on your behalf.
```

Fíjate que el prompt de confirmación aparece solo, en ambos casos — ningún código en `cmd/finance-agent/main.go` lo maneja; `cmd/launcher/console` lo hace por ti. 🙌

### Paso 3: Prueba el Camino de Rechazo 🚫

Corre el mismo comando, pero escribe cualquier cosa distinta de `yes` en el prompt de confirmación (por ejemplo, `no`). **Observa:** el agente reporta que la inversión *no* se procesó, y — gracias a una instrucción explícita en `finance_instruction.md` — no escala al supervisor. Un rechazo es definitivo.

### Paso 4: Lee las Pruebas Automatizadas 🧪

`internal/agents/financeagent/agent_test.go`'s `askInvestment` corre exactamente el mismo ida-y-vuelta de dos turnos de forma programática: envía la solicitud, encuentra la llamada `adk_request_confirmation`, envía de vuelta un `FunctionResponse` sintetizado. `TestInvestment_LargeAmountApproved_{Ollama,Gemini}` verifica que la conversación de verdad termine con `supervisor` como autor — no solo que la herramienta devolvió "escalated" — porque (según el README) la transferencia necesita un turno más del modelo para pasar realmente.

### Troubleshooting

Ve [troubleshooting.md](./troubleshooting.md) si algún paso no se comporta como esperabas.

### Resumen del Laboratorio 🎉

Construiste un agente financiero que se pausa para aprobación humana antes de cada inversión y escala dinámicamente las grandes a un supervisor — usando `RequireConfirmation` y `ctx.Actions().TransferToAgent`, sin necesitar ningún wrapper `Workflow`. ¡Buen trabajo!

### Preguntas de Autorreflexión 🤔
- ¿Por qué debe `RequireConfirmation` estar impuesto por el framework en vez de dejárselo a las instrucciones del LLM?
- La transferencia de escalación necesita un turno extra del modelo para pasar de verdad. ¿Qué saldría mal si tus pruebas asumieran que pasa en el evento inmediatamente siguiente?
- ¿Qué agregarías a `finance_instruction.md` si quisieras que el agente también confirme *en qué* se está invirtiendo, no solo el monto?

<hr/>

> **¿Vienes de Python?** 🐍 El laboratorio de Python te hace completar dos `TODO`s: envolver la herramienta en `FunctionTool(..., require_confirmation=True)` y poner `tool_context.actions.transfer_to_agent = "supervisor"`. Ambos mapean directamente al código Go de este laboratorio. El wrapper `Workflow(edges=[("START", finance_agent)])` de Python no tiene equivalente aquí — el `finance_agent` de este laboratorio es un `llmagent` plano con `SubAgents`, confirmado suficiente.
