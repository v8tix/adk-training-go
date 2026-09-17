# Laboratorio 2: Reto de Configuración de Entorno (Go) ✅

## Prerrequisitos

Antes de empezar, asegúrate de tener listo:

* **Editor de código (IDE):** [VS Code](https://code.visualstudio.com/) o tu editor preferido con soporte para Go.
* **Go 1.27+** instalado localmente. Este repo todavía no tiene soporte para Codespaces o DevContainer — la configuración local es el único camino soportado por ahora.

## Objetivo

Tu tarea: verificar que el entorno de Go de este repo está listo para desarrollo con ADK 2.0. Si te atascas, trabaja hacia atrás desde la sección de Solución de Problemas más abajo. 🕵️

## Tareas del Laboratorio

### Paso 0: Confirma Go 1.27+ (Crucial) 🔍

Revisa tu versión instalada:

```bash
go version
```

Si reporta algo por debajo de `go1.27`, consigue una versión más nueva de Go en `https://go.dev/dl/` antes de continuar.

### Paso 1: Confirma la Dependencia de ADK

Desde la raíz del repo, confirma que `google.golang.org/adk/v2` es una dependencia declarada:

```bash
grep adk go.mod
```

Si falta (por ejemplo, si estás trabajando desde un punto anterior en la historia del repo), agrégala tú mismo:

```bash
go get google.golang.org/adk/v2
go mod tidy
```

### Paso 2: Configura la Autenticación (Opcional)

Por defecto, `cmd/verify-setup` llama a un servidor Ollama local — no necesitas archivo `.env`. ¿Quieres verificar credenciales de Google Cloud/AI Studio en su lugar?

1. Copia `.env.example` a `.env` (ya está cubierto por `.gitignore` — nunca subas el archivo real).
2. Configura tu clave y cambia de camino:
   ```
   GOOGLE_AI_STUDIO_API_KEY="YOUR_API_KEY"
   MODEL_TYPE=gemini
   ```

### Paso 3: Lee `cmd/verify-setup` 📖

Abre `cmd/verify-setup/main.go` y `checks.go`, además de `internal/infrastructure/llm/config.go` y `factory.go` (movidos allí en el módulo 3 para que un segundo programa pudiera reutilizarlos — revisa la documentación de ese módulo para saber por qué). Trata de encontrar:

1. La función que verifica la versión resuelta de `google.golang.org/adk/v2` — ¿cómo la lee sin importar un paquete `internal` al que no tiene permitido acceder?
2. La función que verifica la versión de Go.
3. Dónde se construye el modelo local de Ollama versus dónde se construye Gemini, y qué decide cuál corre.
4. De dónde salen los valores por defecto de `OLLAMA_BASE_URL`, `OLLAMA_MODEL`, y las demás variables configurables por `.env`.

### Paso 4: Corre la Verificación 🚀

```bash
go run ./cmd/verify-setup
```

O, para verificar el camino de Gemini en su lugar:

```bash
MODEL_TYPE=gemini go run ./cmd/verify-setup
```

### 💡 Solución de problemas: Conexión rechazada

Si ves un error de conexión a `localhost:11434`, el servidor Ollama local no está corriendo — confirma que está activo (`ollama list`), o vuelve a `MODEL_TYPE=gemini` con un `.env` configurado.

## Preguntas de Autorreflexión 🤔

* El toolchain de Go incluye de fábrica lo que Python necesita `uv` para hacer (bloqueo de dependencias, builds reproducibles). ¿Cuál es el trade-off de que eso sea una característica del lenguaje en vez de una herramienta aparte?
* ¿Por qué `cmd/verify-setup` lee la versión del SDK de ADK vía `runtime/debug.ReadBuildInfo()` en vez de importar directamente el paquete de versión del propio SDK?
* ¿Cuáles son las implicaciones de seguridad de usar por defecto un modelo local sin API key, versus requerir credenciales de nube desde el principio?

<hr/>

### ¿Buscas la solución? 🔍

Pista: lee la función `BuildModel` en `internal/infrastructure/llm/factory.go` y el mapa `modelFactories` — eso es lo que decide entre Ollama y Gemini, según la variable de entorno `MODEL_TYPE`.
