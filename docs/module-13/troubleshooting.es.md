# Troubleshooting: Módulo 13 (Go) 🔧

### El prompt de confirmación nunca aparece

**Causa:** `functiontool.Config.RequireConfirmation` no está realmente en `true` en la herramienta pasada a `Tools` — las instrucciones del LLM no tienen ningún efecto acá, es una puerta a nivel de framework, no algo que un prompt pueda pedir o saltarse.

**Solución:** verifica que `RequireConfirmation: true` esté puesto en el `functiontool.Config` pasado a `functiontool.New`, no solo mencionado en la descripción de la herramienta.

### Aprobar una inversión grande nunca llega a `supervisor`

**Causa:** `finance_instruction.md` no le dice explícitamente al modelo que llame él mismo a la herramienta de transferencia cuando el estado es "escalated." Sin esa línea, un modelo puede simplemente narrar la escalación en texto sin llegar a llamar realmente a `transfer_to_agent` — confirmado en vivo durante la construcción de este módulo. ¡Un modo de falla escurridizo! 😬

**Solución:** asegúrate de que la instrucción del agente financiero diga explícitamente que debe llamar a `transfer_to_agent` cuando el resultado sea "escalated," no solo describir que la escalación debería pasar.
