# Troubleshooting: Módulo 14 (Go) 🔧

### `unable to fetch the results` 😤

**Causa:** las llamadas subyacentes a la API de Wikipedia de `github.com/trietmn/go-wiki` fallan con este error real si no defines un User-Agent distintivo — Wikimedia limita la tasa del valor por defecto genérico del paquete, ya que lo comparten todos los usuarios del paquete. Confirmado en vivo.

**Solución:** llama a `gowiki.SetUserAgent("your-app-name/1.0 (contact-info)")` una vez, antes de cualquier solicitud:

```go
func init() {
    gowiki.SetUserAgent("your-app-name/1.0 (contact-info)")
}
```

`internal/agents/factfinder/tools.go` hace esto en su propio `init()` de paquete, confirmado en vivo que se respeta en llamadas posteriores hechas desde una función distinta — el clásico patrón de "define una vez, en cualquier lugar antes del primer uso" que siempre necesita un arreglo a nivel de paquete. ¿Sigues viendo el error después de confirmar que el `init()` de `tools.go` realmente corre (revisa que esté realmente definido, no solo documentado)? Espera unos segundos y reintenta — Wikimedia a veces limita rangos de IP compartidos o de la nube sin importar un User-Agent bien definido, un problema distinto y transitorio.
