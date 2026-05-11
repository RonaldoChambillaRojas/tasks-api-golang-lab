# tasks-api-golang-lab

API REST de tareas construida en Go como proyecto de aprendizaje.
Stack de referencia: TypeScript / NestJS.

---

## Tabla de contenidos

1. [¿Qué hace este proyecto?](#qué-hace-este-proyecto)
2. [Cómo correrlo](#cómo-correrlo)
3. [Estructura del proyecto](#estructura-del-proyecto)
4. [¿La estructura es correcta?](#la-estructura-es-correcta)
5. [Comparación con NestJS](#comparación-con-nestjs)
6. [El Dockerfile explicado](#el-dockerfile-explicado)
7. [Conceptos clave de Go (resumen rápido)](#conceptos-clave-de-go-resumen-rápido)

---

## ¿Qué hace este proyecto?

Una API CRUD de tareas con almacenamiento en memoria (sin base de datos).
Endpoints disponibles:

| Método | Path           | Descripción               |
|--------|----------------|---------------------------|
| GET    | /tasks         | Listar todas las tareas   |
| POST   | /tasks         | Crear una tarea           |
| GET    | /tasks/{id}    | Obtener una tarea por ID  |
| PUT    | /tasks/{id}    | Actualizar una tarea      |
| DELETE | /tasks/{id}    | Eliminar una tarea        |

Ejemplo de creación:
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "aprender Go"}'
```

---

## Cómo correrlo

**Sin Docker (desarrollo local):**
```bash
go run .
```

**Con Docker:**
```bash
docker build -t tasks-api .
docker run -p 8080:8080 tasks-api
```

**Compilar el binario manualmente:**
```bash
go build -o tasks-api .
./tasks-api
```

---

## Estructura del proyecto

```
tasks-api-golang-lab/
├── main.go           # Punto de entrada: configura rutas y arranca el servidor
├── go.mod            # Equivale a package.json — define el módulo y dependencias
├── go.sum            # Equivale a package-lock.json — hashes de dependencias
├── dockerfile        # Build multi-stage para producción
│
├── model/
│   └── task.go       # Struct Task (entidad/DTO del dominio)
│
├── handler/
│   └── task.go       # Handlers HTTP (equivale a un Controller de NestJS)
│
└── store/
    └── task.go       # Almacenamiento en memoria (equivale a un Repository)
```

### ¿Qué es cada cosa?

| Archivo | Rol en Go | Equivalente NestJS |
|---|---|---|
| `model/task.go` | Define la estructura de datos | Entity / DTO |
| `store/task.go` | Acceso y mutación de datos | Repository / Service (capa de datos) |
| `handler/task.go` | Recibe request, llama al store, responde | Controller |
| `main.go` | Crea dependencias y registra rutas | main.ts (bootstrap) + módulo raíz |

---

## ¿La estructura es correcta?

**Sí, para un proyecto de aprendizaje es correcta y limpia.**
Las capas están bien separadas y el flujo de datos es claro.

Sin embargo, hay algunas diferencias con una estructura de proyecto Go más madura:

### Lo que está bien

- Separación en paquetes por responsabilidad (`model`, `handler`, `store`).
- Inyección de dependencias explícita y manual — nada de magia.
- El store protege su map con `sync.RWMutex` — correcto para concurrencia.
- Los handlers son funciones puras sin estado global.

### Mejoras para un proyecto productivo

**1. Usar una interfaz para el store.**

Actualmente el handler depende directamente de `*store.TaskStore` (tipo concreto).
En Go, la buena práctica es depender de interfaces, no de implementaciones:

```go
// En handler/task.go, en lugar de:
type TaskHandler struct {
    store *store.TaskStore
}

// Definir una interfaz:
type TaskStorer interface {
    GetAll() []model.Task
    GetByID(id string) (model.Task, error)
    Create(title string) model.Task
    Update(id string, title string, done bool) (model.Task, error)
    Delete(id string) error
}

type TaskHandler struct {
    store TaskStorer  // depende de la interfaz, no del tipo concreto
}
```

Esto permite intercambiar el store en memoria por uno que hable con PostgreSQL
sin cambiar una sola línea del handler. También facilita los tests con mocks.

**2. Estructura estándar para proyectos más grandes.**

Para proyectos con más features, la comunidad Go recomienda:

```
myapp/
├── cmd/
│   └── api/
│       └── main.go       # el binario vive aquí
├── internal/             # código privado al módulo (no importable externamente)
│   ├── handler/
│   ├── store/
│   └── model/
├── pkg/                  # código reutilizable que otros módulos pueden importar
└── go.mod
```

Para este tamaño de proyecto, la estructura plana actual es perfectamente válida.

**3. Un detalle en go.mod.**

Chi está marcado como `// indirect` en go.mod, pero es una dependencia directa.
Esto no afecta el funcionamiento, pero es impreciso. `go mod tidy` lo corrige:

```bash
go mod tidy
```

---

## Comparación con NestJS

| Concepto | NestJS | Este proyecto (Go + chi) |
|---|---|---|
| Framework | NestJS (opinionado, completo) | chi (solo router, minimalista) |
| Routing | Decoradores `@Get()`, `@Post()` | Registro explícito en `main.go` |
| DI Container | Automático (módulos/providers) | Manual: `NewXxx(dep)` |
| Validación | `class-validator` + DTOs | Manual (o `go-playground/validator`) |
| Error handling | Exceptions + ExceptionFilters | Múltiples valores de retorno `(T, error)` |
| Concurrencia | Single-thread + event loop | Goroutines (hilos ligeros reales) |
| Tipado | TypeScript (en compile time) | Go (fuertemente tipado, en compile time) |
| Compilación | TS → JS (necesita Node en runtime) | Go → binario nativo (no necesita runtime) |

### Flujo de una request: NestJS vs este proyecto

**NestJS:**
```
Request → Middleware → Guard → Interceptor → Pipe (validación) → Controller → Service → Repository → DB
```

**Este proyecto:**
```
Request → middleware.Logger → middleware.Recoverer → Handler → Store → map en memoria
```

Ambos siguen el mismo principio de capas, pero en Go tú controlas cada pieza manualmente.

---

## El Dockerfile explicado

```dockerfile
# Stage 1: compilar el binario
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o tasks-api .

# Stage 2: imagen final vacía
FROM scratch

COPY --from=builder /app/tasks-api /tasks-api

EXPOSE 8080

ENTRYPOINT ["/tasks-api"]
```

### ¿Por qué dos stages (multi-stage build)?

El build multi-stage separa la etapa de **compilación** de la etapa de **ejecución**.

- **Stage 1 (`builder`)**: necesita todo el toolchain de Go para compilar.
  La imagen `golang:1.26-alpine` pesa ~300MB — herramientas, compilador, librerías.
- **Stage 2 (`scratch`)**: solo lleva el binario compilado.

El resultado final **no incluye** el compilador, el código fuente, ni el sistema operativo.
Solo el binario ejecutable.

### La gran diferencia con NestJS: `FROM scratch`

Esta es la diferencia más impactante respecto a NestJS (o cualquier app Node.js):

**Dockerfile típico de NestJS:**
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine          # ← necesita Node.js en la imagen final
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
CMD ["node", "dist/main.js"]
```

**Este proyecto Go:**
```dockerfile
FROM scratch                 # ← imagen vacía, literalmente nada
COPY --from=builder /app/tasks-api /tasks-api
ENTRYPOINT ["/tasks-api"]
```

**¿Por qué Go puede usar `scratch` y Node.js no puede?**

- **Node.js** interpreta JavaScript: el runtime de Node.js (`node`) debe estar
  presente en la imagen para ejecutar el código. No existe "compilar a binario
  nativo" en Node — siempre necesitas el intérprete.

- **Go** compila a un binario nativo del sistema operativo. El binario resultante
  contiene TODO lo necesario para ejecutarse: lógica de la app, stdlib de Go,
  scheduler de goroutines. No necesita ningún runtime externo.

### Las flags de compilación explicadas

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o tasks-api .
```

| Flag | Qué hace |
|---|---|
| `CGO_ENABLED=0` | Deshabilita CGO (bindings con C). Sin esto, el binario podría depender de `libc` del OS, haciendo imposible usar `scratch`. Con CGO=0 el binario es 100% estático. |
| `GOOS=linux` | Cross-compila para Linux. Puedes buildear desde macOS o Windows y el binario correrá en Linux del container. |
| `GOARCH=amd64` | Target architecture: x86-64 (la mayoría de servidores). En ARM uses `arm64`. |
| `-ldflags="-w -s"` | Strips de debug: `-w` elimina la tabla DWARF (debug info) y `-s` elimina la symbol table. Reduce el tamaño del binario ~30%. |
| `-o tasks-api` | Nombre del binario de salida. |

### Comparación de tamaño de imágenes finales

| Stack | Imagen base final | Tamaño aproximado |
|---|---|---|
| NestJS | `node:20-alpine` | ~200-400 MB |
| NestJS (optimizado) | `node:20-alpine` (solo dist + deps de prod) | ~150-250 MB |
| Este proyecto Go | `scratch` | ~8-15 MB |

El binario Go es pequeño porque no arrastra un runtime, intérprete, ni `node_modules`.

### ¿Cuándo NO usar `scratch`?

`scratch` es ideal cuando no necesitas nada del OS. Si tu app necesita:
- Certificados TLS (HTTPS directo sin proxy): usa `alpine` y copia los certs de `/etc/ssl`
- Variables de entorno de sistema, usuarios, shells: usa `alpine` o `distroless`
- Acceso a DNS dinámico (resolver): copia `/etc/resolv.conf` o usa `distroless`

Para esta API que no hace llamadas HTTPS externas, `scratch` es perfecto.

---

## Conceptos clave de Go (resumen rápido)

Para alguien que viene de TypeScript/NestJS, estos son los conceptos que más sorprenden:

### 1. Visibilidad por mayúscula, no por keyword
```go
type Task struct {
    ID    string  // exportado (public) — mayúscula
    title string  // no exportado (private) — minúscula
}
```

### 2. Múltiples valores de retorno en lugar de excepciones
```go
// En lugar de try/catch, las funciones devuelven (resultado, error)
task, err := store.GetByID(id)
if err != nil {
    // manejo del error
}
```

### 3. Struct + receiver en lugar de clases
```go
// No hay clases. Los métodos se "adjuntan" a tipos con receivers.
func (s *TaskStore) GetAll() []model.Task {
    // s es como "this"
}
```

### 4. Punteros explícitos
```go
s := store.NewTaskStore()    // *TaskStore (puntero — comparte estado)
var t model.Task             // model.Task (valor — copia independiente)
```

### 5. defer — cleanup garantizado
```go
s.mu.Lock()
defer s.mu.Unlock()  // siempre se ejecuta al final, sin importar cómo salga la función
```

### 6. Interfaces implícitas
En Go no declaras `implements MiInterfaz`. Si un tipo tiene los métodos requeridos, ya la implementa. El compilador lo verifica automáticamente. Esto es duck typing estático.

### 7. Concurrencia con goroutines
```go
go miFuncion()  // lanza miFuncion en un goroutine (hilo ligero)
```
Go maneja cada request HTTP en su propio goroutine automáticamente. Por eso el mutex en el store es necesario — sin él, dos requests podrían escribir al mismo map simultáneamente.
