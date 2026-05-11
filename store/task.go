// Package store implementa la capa de persistencia/acceso a datos.
// En NestJS esto equivale a un Repository (con TypeORM) o un Service
// que encapsula la lógica de acceso a datos. Aquí usamos un map en memoria,
// pero la "interfaz" del store sería la misma si conectaras a PostgreSQL.
//
// En Go no existe inyección de dependencias automática ni un ORM por defecto.
// Tú construyes el store y lo pasas manualmente a quien lo necesite.
package store

import (
	// "errors" provee errors.New() para crear valores de error simples.
	//
	// CONCEPTO FUNDAMENTAL: en Go los errores son VALORES, no excepciones.
	// No existe try/catch. Las funciones que pueden fallar devuelven un
	// segundo valor de tipo "error". El llamador DEBE revisarlo.
	// La interfaz "error" es simplemente:
	//   type error interface { Error() string }
	// Cualquier tipo que tenga un método Error() string implementa error.
	"errors"

	// "fmt" es el paquete de formateo (Format). Provee Sprintf, Println, etc.
	// Fmt.Sprintf formatea strings con verbos como %d (int), %s (string), %v (any).
	// Equivale a los template literals de JS: `${valor}` → fmt.Sprintf("%v", valor)
	"fmt"

	// "sync" provee primitivas de sincronización para concurrencia.
	//
	// DIFERENCIA CLAVE CON NODE.JS:
	// Node.js es single-threaded con event loop — solo un hilo ejecuta JS.
	// Go puede ejecutar múltiples goroutines en PARALELO en distintos núcleos.
	// Esto significa que dos requests pueden llegar simultáneamente y acceder
	// al mismo map al mismo tiempo → corrupción de datos sin sincronización.
	//
	// sync.RWMutex (Read-Write Mutex) resuelve esto:
	//   RLock/RUnlock  → múltiples lectores simultáneos permitidos
	//   Lock/Unlock    → un solo escritor, bloquea a todos los demás
	"sync"

	// Importamos nuestro propio paquete model usando el module name
	// definido en go.mod ("tasks-api") + el subdirectorio ("model").
	// En TypeScript sería: import { Task } from '../model/task'
	// En Go se importa el PAQUETE completo, no exportaciones individuales.
	// Se accede como: model.Task, model.OtroTipo, etc.
	"tasks-api/model"
)

// TaskStore es la estructura que representa nuestro "repositorio" en memoria.
// En NestJS con TypeORM sería algo como:
//
//	@Injectable()
//	export class TaskRepository {
//	  constructor(
//	    @InjectRepository(Task)
//	    private readonly repo: Repository<Task>,
//	  ) {}
//	}
//
// En Go no hay decoradores ni contenedor de IoC. TaskStore es un struct plano
// con sus campos. La "inyección" la hacemos manualmente en main.go.
type TaskStore struct {
	// mu es el mutex para proteger accesos concurrentes al map.
	// Nombre "mu" es convención de la comunidad Go (como "err" para errores).
	// Empieza con minúscula → privado al paquete store (no accesible desde fuera).
	//
	// sync.RWMutex se puede usar como zero value (sin inicializar):
	// un mutex no inicializado es un mutex desbloqueado válido.
	mu sync.RWMutex

	// tasks es el almacén en memoria: un map de Go.
	// map[string]model.Task → "un map donde las claves son string y los valores son Task"
	// Equivale en TypeScript a: Map<string, Task> o Record<string, Task>
	//
	// IMPORTANTE: los maps en Go NO son seguros para uso concurrente.
	// Por eso necesitamos el mutex arriba — sin él, accesos paralelos
	// al mismo map pueden causar panic o corrupción silenciosa.
	tasks map[string]model.Task

	// counter es un entero para generar IDs autoincrementales simples.
	// "int" en Go es un entero con signo cuyo tamaño depende de la
	// arquitectura (64 bits en sistemas modernos). No existe "number" como
	// en TypeScript — Go tiene tipos numéricos específicos:
	//   int, int8, int16, int32, int64 (enteros con signo)
	//   uint, uint8, uint16, uint32, uint64 (enteros sin signo)
	//   float32, float64 (punto flotante)
	// El zero value de int es 0, así que counter arranca en 0 automáticamente.
	counter int
}

// NewTaskStore es el "constructor" de TaskStore.
//
// En Go NO existen constructores ni clases. La convención idiomática es:
//   New<NombreDelTipo>() *NombreDelTipo
//
// Esta función crea e inicializa el struct y devuelve un puntero a él.
//
// En TypeScript/NestJS el constructor sería:
//
//	constructor() { this.tasks = new Map() }
//
// El tipo de retorno *TaskStore es un "puntero a TaskStore".
// En Go los punteros son EXPLÍCITOS (diferente a JS donde los objetos
// siempre son referencias implícitas). El asterisco (*) en el tipo significa
// "puntero a". El ampersand (&) en el valor significa "dame la dirección de".
func NewTaskStore() *TaskStore {
	// &TaskStore{...} hace dos cosas:
	//   1. Crea un TaskStore con los campos especificados
	//   2. El & devuelve un PUNTERO a ese struct (su dirección de memoria)
	//
	// make(map[string]model.Task) inicializa el map.
	// En Go hay tres "built-in functions" para inicializar tipos compuestos:
	//   make(map[K]V)    → inicializa un map
	//   make([]T, len)   → inicializa un slice
	//   make(chan T)      → inicializa un channel
	// Un map declarado pero NO inicializado vale nil. Escribir en un map nil
	// causa panic en runtime. make() crea las estructuras internas necesarias.
	// Es equivalente a: new Map() en JavaScript.
	return &TaskStore{
		tasks: make(map[string]model.Task),
	}
}

// GetAll devuelve todas las tareas almacenadas como un slice (lista ordenable).
//
// SINTAXIS DEL RECEIVER:
//   func (s *TaskStore) GetAll() []model.Task
//   └─────────────────┘
//   Esto es el "receiver" — le dice a Go que GetAll es un MÉTODO de TaskStore.
//   (s *TaskStore) es como "this: TaskStore" en TypeScript.
//   El nombre "s" es convención: inicial del tipo (TaskStore → s).
//
// []model.Task es un "slice": la estructura de lista dinámica de Go.
// Equivale a Task[] en TypeScript o un array en JS.
// Un slice tiene: longitud (len), capacidad (cap) y un puntero al array subyacente.
//
// En NestJS sería: findAll(): Task[] { return [...this.tasks.values()] }
func (s *TaskStore) GetAll() []model.Task {
	// RLock: adquiere lock de lectura. Permite lecturas simultáneas
	// desde múltiples goroutines, pero bloquea si hay un escritor activo.
	s.mu.RLock()

	// defer ejecuta la expresión cuando la función TERMINA (cualquier return).
	// Es similar a un "finally" de try/finally, pero solo para una expresión.
	//
	// Por qué defer aquí: garantizamos que RUnlock siempre se llame,
	// incluso si el código de abajo paniquea o tiene múltiples returns.
	// Es la forma idiomática de Go para evitar "lock leaks".
	defer s.mu.RUnlock()

	// make([]model.Task, 0, len(s.tasks)) crea un slice con:
	//   - length = 0 → ningún elemento todavía
	//   - capacity = len(s.tasks) → espacio preallocado en memoria
	// La preallocación evita re-allocations del slice mientras hacemos append.
	// Como new Array(n) en JS, pero para slices de Go.
	list := make([]model.Task, 0, len(s.tasks))

	// for-range itera sobre el map. Devuelve par (clave, valor) en cada iteración.
	// Sintaxis: for key, value := range coleccion
	//
	// El _ (blank identifier) descarta la clave del map. En Go NO puedes declarar
	// una variable y no usarla — el compilador falla con "declared and not used".
	// _ es la forma de decirle explícitamente "este valor no me importa".
	//
	// Equivale a: for (const t of this.tasks.values()) en JS.
	for _, t := range s.tasks {
		// append agrega t al slice y devuelve el slice resultante.
		// IMPORTANTE: append puede devolver un NUEVO slice si necesita
		// más capacidad (reallocation). Por eso debes reasignar: list = append(list, t).
		// Es diferente a Array.push() de JS que muta in-place.
		// Si no reasignaras: append(list, t) sin list =, el elemento se perdería.
		list = append(list, t)
	}
	return list
}

// GetByID busca una tarea por su ID y la devuelve.
//
// MÚLTIPLES VALORES DE RETORNO — uno de los features más distintivos de Go:
//   func GetByID(id string) (model.Task, error)
// Las funciones pueden devolver más de un valor. La convención es:
//   (resultado, error) — si error != nil, algo falló; si nil, todo bien.
//
// En TypeScript manejarías esto con:
//   - throw/catch para excepciones
//   - Promise.reject() en async functions
//   - Result<T, E> con librerías como neverthrow o fp-ts
//
// En Go el manejo de errores es EXPLÍCITO y OBLIGATORIO. El llamador
// debe revisar el error — el compilador no te obliga, pero es la convención
// fuertemente seguida en toda la comunidad y librerías de Go.
func (s *TaskStore) GetByID(id string) (model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Acceso a un map en Go devuelve DOS valores: (value, ok bool).
	// "ok" es true si la clave existe, false si no.
	//
	// Si solo usas: t := s.tasks[id], obtienes el zero value de Task{}
	// cuando la clave no existe — sin manera de saber si realmente estaba.
	// La forma de dos valores es la única forma segura de verificar existencia.
	//
	// := es el "short variable declaration": declara Y asigna en una línea.
	// Equivale a: const [t, ok] = ... pero tipado estáticamente.
	t, ok := s.tasks[id]

	if !ok {
		// No encontramos la tarea. Devolvemos:
		//   - model.Task{} → zero value del struct (equivale a {} vacío)
		//   - errors.New("...") → un error con mensaje descriptivo
		//
		// En NestJS: throw new NotFoundException('task not found')
		// En Go: return valor_vacio, errors.New("task not found")
		// El llamador decide qué hacer con el error — no "explota" automáticamente.
		return model.Task{}, errors.New("task not found")
	}

	// Camino feliz: devolvemos la tarea y nil como error.
	// nil en Go es el zero value para: punteros, interfaces, slices, maps, channels, functions.
	// Un error nil significa "no hubo error". El llamador checkea: if err != nil { ... }
	return t, nil
}

// Create crea una nueva tarea con el título dado y la almacena en el map.
// Devuelve la tarea creada como valor (no puntero) — una copia del struct.
func (s *TaskStore) Create(title string) model.Task {
	// Lock() es el lock de escritura: EXCLUSIVO.
	// Bloquea a todos los demás goroutines (lectores Y escritores)
	// hasta que llamemos Unlock(). Solo un escritor a la vez.
	s.mu.Lock()
	defer s.mu.Unlock()

	// ++ en Go es solo postfix (s.counter++), no existe prefix (++s.counter).
	// También: no puedes usar s.counter++ como expresión — solo como statement.
	// Es decir: x := s.counter++ no compila en Go.
	s.counter++

	// fmt.Sprintf formatea un string. Verbos comunes:
	//   %d → entero (decimal)
	//   %s → string
	//   %f → float
	//   %v → "valor por defecto" (funciona con cualquier tipo)
	//   %+v → struct con nombres de campos
	//   %T → tipo del valor
	// Equivale a: `${s.counter}` en TypeScript.
	id := fmt.Sprintf("%d", s.counter)

	// Struct literal: creamos un Task especificando sus campos por nombre.
	// Los campos no listados toman su zero value.
	// En TypeScript: const t: Task = { ID: id, Title: title, Done: false }
	t := model.Task{ID: id, Title: title, Done: false}

	// Asignamos la tarea al map. El map almacena el valor por copia.
	s.tasks[id] = t

	return t
}

// Update modifica título y estado de una tarea existente.
// Devuelve la tarea actualizada o un error si el ID no existe.
func (s *TaskStore) Update(id string, title string, done bool) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, errors.New("task not found")
	}

	// t es una COPIA del valor almacenado en el map (los maps almacenan por valor).
	// Mutamos los campos de la copia local t.
	// Luego debemos guardarlo de vuelta: s.tasks[id] = t
	// Si el map almacenara punteros (*model.Task), podríamos mutar directamente
	// sin este paso — pero aquí usamos valores, no punteros.
	t.Title = title
	t.Done = done

	// Guardamos la copia modificada de vuelta en el map.
	s.tasks[id] = t
	return t, nil
}

// Delete elimina una tarea del map por su ID.
// Devuelve nil si tuvo éxito, o error si la tarea no existía.
// Nota: aquí solo devolvemos error (sin valor de tarea) porque DELETE
// no necesita retornar el recurso eliminado.
func (s *TaskStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Usamos _, ok para verificar existencia sin guardar el valor de la tarea.
	// El scope de "ok" queda limitado al bloque if — Go usa scoping léxico
	// igual que TypeScript con let/const.
	if _, ok := s.tasks[id]; !ok {
		return errors.New("task not found")
	}

	// delete() es una built-in function de Go para eliminar una clave de un map.
	// Equivale a: this.tasks.delete(id) en JS (Map) o delete obj[id] en JS (objeto).
	// delete() en un map no falla si la clave no existe — por eso verificamos antes.
	delete(s.tasks, id)

	// nil como error = "todo salió bien, no hay error que reportar"
	return nil
}
