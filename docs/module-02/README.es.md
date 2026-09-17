# Módulo 2: Configura tu Entorno de Desarrollo (Go) 🛠️

## Teoría

### Por qué importa tener un entorno limpio

Antes de construir agentes, dejemos bien configurado tu entorno de desarrollo. Un entorno bien armado mantiene las dependencias de tu proyecto aisladas — sin conflictos con otros proyectos de Go en tu máquina — y hace que todo sea autocontenido y fácil de reproducir para cualquier otra persona. 🧹

### Go 1.27+ y ADK 2.0: lo que necesitas

* **Go:** estrictamente **1.27 o superior** — ese es el piso elegido para este curso (coincide con `go.mod`).
* **ADK:** `google.golang.org/adk/v2` (`v2.4.0`+), el SDK oficial de Go para ADK 2.0. Dato curioso: su propio requisito de paquete es Go 1.25+ — más bajo que el piso de este curso, ya que 1.27 es lo que este repo realmente usa.

### Módulos de Go: gestión de dependencias, incluida de fábrica 📦

El toolchain de Go maneja directamente la gestión de módulos, el bloqueo de dependencias y los builds reproducibles — sin necesidad de un gestor de paquetes aparte:

* `go.mod` declara tu módulo y sus dependencias.
* `go.sum` fija las versiones exactas de las dependencias y sus checksums.
* `go get <module>` agrega una dependencia; `go mod tidy` mantiene `go.mod`/`go.sum` sincronizados con lo que tu código realmente importa.

Tampoco hay un entorno virtual aparte que activar — un módulo de Go *es* tu unidad aislada y reproducible. Lindo, ¿no?

<hr/>

> **¿Vienes de Python?** 🐍 `go.mod` cumple el rol del archivo de proyecto de `uv` (o `pip`), y `go.sum` es el equivalente de Go a `uv.lock` — pero no hay una herramienta aparte que instalar primero; todo viene incluido en el comando `go` que ya tienes.

### Flujo de trabajo de desarrollo

Este repo actualmente solo soporta desarrollo local en Go:

1. Instala Go 1.27+ desde `https://go.dev/dl/`.
2. Verifícalo: `go version`.
3. Desde la raíz del repo, las dependencias ya están declaradas en `go.mod` — `go build ./...` las descarga y construye todo.

> Codespaces y una configuración `.devcontainer` todavía no están disponibles para este repo — una configuración de un clic en el navegador es trabajo futuro, no parte de este módulo.

### Autenticación: Conectando con un Modelo 🔑

Este curso usa por defecto un modelo **local** — no necesitas credenciales de nube para empezar. 🎉

#### Opción A: Ollama Local (Por defecto, recomendado para este curso)

`cmd/verify-setup` y todos los módulos siguientes llaman a un servidor Ollama local por defecto: `http://localhost:11434` (endpoint `/v1` compatible con OpenAI), modelo `qwen3.8:27b` — una cuantización GGUF, elegida porque es la que se confirmó (módulo 4) que soporta salida estructurada restringida por esquema JSON, además de texto plano. No necesitas API key ni entrada en `.env` para este camino — Ollama no verifica el valor de la clave. Todo esto es configurable: copia `.env.example` a `.env` y sobrescribe `OLLAMA_BASE_URL` / `OLLAMA_MODEL` si tu servidor Ollama vive en otro lado.

#### Opción B: API Key de Google AI Studio

1. Consigue una API key en Google AI Studio: `https://aistudio.google.com/app/apikey`.
2. Copia `.env.example` a `.env` y define `GOOGLE_AI_STUDIO_API_KEY="YOUR_API_KEY"` y `MODEL_TYPE=gemini` — esto cambia `cmd/verify-setup` (y todo lo que sigue) a este camino.

#### Opción C: Autenticación de Google Cloud (Enterprise)

Para producción, usa Application Default Credentials vía el CLI de Google Cloud:

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### Solución de errores comunes 🔧

#### 1. 🛑 El build falla con "no required module provides package ..."

Corre `go mod tidy` — resuelve y fija cualquier dependencia que tu código importe pero que `go.mod` todavía no declare.

#### 2. 🛑 `404: Model not found` (solo camino Gemini)

Generalmente significa que el modelo aún no fue desplegado en tu ubicación de Google Cloud. Cambia `GOOGLE_CLOUD_LOCATION` (o `LOCATION` en `.env`) a alguna de: `us-central1`, `us-east4`, `europe-west9`.

#### 3. 🛑 `PermissionDenied: 403` (solo camino Gemini)

A tu cuenta le falta el rol "Agent Platform User". Otórgalo vía IAM en la Consola de Cloud (`roles/aiplatform.user`).

#### 4. 🛑 Conexión rechazada a `localhost:11434` (camino Ollama)

El servidor Ollama local no está corriendo, o `OLLAMA_BASE_URL` en tu `.env` apunta a algo inalcanzable. Confirma que Ollama está activo (`ollama list`), o cambia al camino Gemini (`MODEL_TYPE=gemini`) con la Opción B o C de arriba.

### Puntos clave ✅

- **Go 1.27+** (el piso de este curso) y `google.golang.org/adk/v2` (`v2.4.0`+) son estrictamente obligatorios.
- El propio toolchain de Go (`go.mod`/`go.sum`/`go mod tidy`) maneja directamente la gestión de dependencias — sin gestor de paquetes extra.
- Este curso usa por defecto un **modelo Ollama local**; solo recurre a un archivo `.env` y `MODEL_TYPE=gemini` cuando necesites verificar credenciales específicas de Google Cloud/AI Studio.
