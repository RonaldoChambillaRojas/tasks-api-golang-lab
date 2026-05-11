// Package handler contiene los manejadores (handlers) HTTP de la API.
// En NestJS esto equivale a un @Controller: recibe la request HTTP,
// delega la lógica al Service/Repository y devuelve la response.
//
// DIFERENCIA ARQUITECTÓNICA:
// En NestJS, el controller declara sus rutas con decoradores (@Get, @Post...)
// y el framework las registra automáticamente al arrancar.
// En Go con chi, el handler es solo una función. Las rutas se registran
// EXPLÍCITAMENTE en main.go — no hay magia automática.
//
// Un handler válido para net/http tiene la firma:
//   func(w http.ResponseWriter, r *http.Request)
// Chi extiende esto con parámetros de ruta ({id}), pero la firma es idéntica.
package handler

import (
	// "encoding/json" provee serialización y deserialización JSON.
	// En Express/NestJS esto lo hace el framework automáticamente:
	// el body se parsea y res.json() serializa. En Go lo hacemos explícitamente.
	"encoding/json"

	// "net/http" es la librería HTTP estándar de Go. Contiene:
	//   http.ResponseWriter → interfaz para escribir la respuesta (como "res" en Express)
	//   *http.Request       → struct con la request (como "req" en Express)
	//   http.StatusOK, http.StatusNotFound, etc. → constantes de códigos HTTP
	"net/http"

	// chi es el router. chi.URLParam() extrae parámetros de ruta como {id}.
	// En Express: req.params.id
	// En NestJS: @Param('id') id: string
	// En chi:    chi.URLParam(r, "id")
	"github.com/go-chi/chi/v5"

	// Importamos el store para tener acceso al "repositorio".
	// Solo el tipo *store.TaskStore — no una instancia global.
	"tasks-api/store"
)

// TaskHandler agrupa todos los handlers HTTP relacionados con la entidad Task.
// En NestJS sería el @Controller('tasks') con todos sus métodos @Get/@Post/etc.
//
// Al ser un struct, puede llevar estado (dependencias inyectadas).
// El campo store es un PUNTERO a TaskStore: todos los handlers comparten
// la misma instancia del store en memoria.
// Si fuera store.TaskStore (sin *), cada método recibiría una COPIA del store
// — los cambios no persistirían entre requests. Los punteros son esenciales
// cuando necesitas compartir estado mutable.
type TaskHandler struct {
	store *store.TaskStore
}

// NewTaskHandler crea una instancia de TaskHandler con el store inyectado.
//
// Esto es "Dependency Injection manual" — en NestJS el framework resuelve
// y provee las dependencias automáticamente mediante el sistema de módulos.
// En Go lo hacemos nosotros: pasamos las dependencias como argumentos y
// las almacenamos en el struct.
//
// Patrón: NewXxx(deps...) *Xxx
//
// En TypeScript con NestJS el equivalente es el constructor del provider:
//
//	constructor(private readonly taskStore: TaskStore) {}
func NewTaskHandler(s *store.TaskStore) *TaskHandler {
	// Struct literal con campo nombrado. El & crea un puntero al nuevo struct.
	return &TaskHandler{store: s}
}

// createRequest define la forma del body JSON esperado para crear una tarea.
//
// En NestJS usarías un DTO con class-validator:
//
//	export class CreateTaskDto {
//	  @IsString()
//	  @IsNotEmpty()
//	  title: string;
//	}
//
// En Go es un struct con struct tags. La validación se hace manualmente
// (o con github.com/go-playground/validator para casos más complejos).
// Empieza con minúscula → privado al paquete handler, no visible desde fuera.
type createRequest struct {
	Title string `json:"title"`
}

// updateRequest define el body JSON esperado para actualizar una tarea.
// Ambos campos son requeridos en este diseño (PUT completo, no PATCH parcial).
type updateRequest struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// writeJSON es una función helper interna que serializa un valor Go a JSON
// y lo escribe en la ResponseWriter con el status code apropiado.
//
// El nombre empieza con minúscula → privada al paquete handler.
// Solo los handlers de este paquete pueden usarla.
//
// Parámetros:
//   - w http.ResponseWriter → el objeto de respuesta (equivale a "res" en Express)
//   - status int            → código HTTP: 200, 201, 404, etc.
//   - v any                 → el valor a serializar como JSON.
//
// "any" es un alias de "interface{}" introducido en Go 1.18.
// interface{} (o any) es el tipo más genérico de Go: cualquier valor
// lo satisface. Equivale a "unknown" en TypeScript — pero sin type guards
// explícitos ya que json.Encode usa reflection internamente para serializar.
func writeJSON(w http.ResponseWriter, status int, v any) {
	// Los headers DEBEN establecerse ANTES de llamar a WriteHeader().
	// Una vez enviado el status code, los headers se vuelven inmutables.
	// En Express/NestJS el framework maneja este orden automáticamente.
	// En Go debemos ser explícitos y respetar el orden.
	w.Header().Set("Content-Type", "application/json")

	// WriteHeader envía el status code HTTP. Solo puede llamarse UNA vez.
	// Llamarlo de nuevo no tiene efecto (pero produce un warning en logs).
	// Si no llamas WriteHeader, Go envía 200 OK automáticamente al primer Write.
	w.WriteHeader(status)

	// json.NewEncoder(w) crea un encoder que escribe DIRECTAMENTE al ResponseWriter.
	// .Encode(v) serializa v a JSON y lo escribe en el stream de respuesta.
	//
	// ALTERNATIVA menos eficiente:
	//   data, _ := json.Marshal(v)   // serializa a []byte en memoria
	//   w.Write(data)                // luego lo escribe
	//
	// json.NewEncoder hace streaming directo: evita crear un buffer intermedio.
	// Es el patrón idiómatico cuando tienes un io.Writer disponible.
	json.NewEncoder(w).Encode(v)
}

// GetAll maneja GET /tasks — devuelve todas las tareas.
//
// ANATOMÍA DEL MÉTODO:
//   func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request)
//   └── (h *TaskHandler) → receiver: le da acceso a h.store (como "this")
//                        → *TaskHandler: puntero, no copia — importante para
//                          acceder al estado compartido del handler
//   w http.ResponseWriter → para escribir la respuesta
//   r *http.Request       → la request con headers, body, query params, etc.
//
// Equivalente NestJS:
//   @Get()
//   getAll(): Task[] {
//     return this.taskService.findAll();
//   }
func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.GetAll())
}

// GetByID maneja GET /tasks/{id} — devuelve una tarea por su ID.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// chi.URLParam extrae el valor del parámetro de ruta registrado como "{id}".
	// La ruta en main.go es: r.Get("/tasks/{id}", h.GetByID)
	// Si la URL es /tasks/42, chi.URLParam(r, "id") devuelve "42".
	//
	// Equivalencias:
	//   Express: req.params.id
	//   NestJS:  @Param('id') id: string
	//   chi:     chi.URLParam(r, "id")
	id := chi.URLParam(r, "id")

	// GetByID devuelve (Task, error) — capturamos ambos valores.
	// Si err != nil → la tarea no existe → respondemos 404.
	// El patrón "if err != nil" es el error handling idiomático de Go.
	// No existe try/catch — el flujo de control es lineal y explícito.
	task, err := h.store.GetByID(id)
	if err != nil {
		// map[string]string{"error": err.Error()} es un map literal inline.
		// Equivale a: { error: err.Error() } en JS.
		// err.Error() llama al método Error() de la interfaz error → string del mensaje.
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		// return termina el handler aquí. El código de abajo NO ejecuta.
		// En Go no hay else implícito después de un if con return — el compilador
		// sabe que si llegamos aquí, el if no se ejecutó. Es el "guard clause" pattern.
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// Create maneja POST /tasks — crea una nueva tarea.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	// "var req createRequest" declara la variable con su zero value: {Title: ""}.
	// Alternativa equivalente: req := createRequest{}
	// Usamos "var" cuando queremos ser explícitos en la declaración sin asignar
	// un valor específico — es una preferencia de estilo.
	var req createRequest

	// json.NewDecoder(r.Body) crea un decoder que lee el body de la request.
	// .Decode(&req) deserializa el JSON y lo asigna a los campos de req.
	//
	// &req pasa un PUNTERO a req. Sin &, Decode recibiría una copia de req
	// y los campos deserializados se perderían — req quedaría vacío.
	// Este es el patrón fundamental en Go: pasar punteros cuando quieres
	// que una función modifique tu variable.
	//
	// En NestJS/Express esto es automático:
	//   @Body() body: CreateTaskDto → el framework parsea y valida
	// En Go es manual y explícito.
	//
	// Aquí también validamos: si Decode falla (JSON malformado) O si Title
	// está vacío (string == "" es el zero value) → 400 Bad Request.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	task := h.store.Create(req.Title)

	// http.StatusCreated es la constante para 201 Created.
	// La librería net/http exporta constantes para todos los códigos HTTP
	// siguiendo la convención http.StatusXxx.
	writeJSON(w, http.StatusCreated, task)
}

// Update maneja PUT /tasks/{id} — actualiza una tarea existente.
// Este es un PUT completo (reemplaza todos los campos), no un PATCH parcial.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// store.Update devuelve (Task, error). Si el id no existe → error → 404.
	task, err := h.store.Update(id, req.Title, req.Done)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// Delete maneja DELETE /tasks/{id} — elimina una tarea.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Cuando la función devuelve solo un error (sin valor de resultado),
	// podemos declarar y checkear err en el mismo if:
	//   if err := expr; condicion { ... }
	// Esto limita el SCOPE de "err" al bloque del if — no es accesible afuera.
	// En Go el scope de variables es preciso y el compilador rechaza variables
	// declaradas y no usadas (error de compilación, no warning).
	if err := h.store.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// 204 No Content: operación exitosa sin body de respuesta.
	// Solo enviamos el status — no llamamos a Encode/writeJSON.
	w.WriteHeader(http.StatusNoContent)
}
