// Package main es el paquete de entrada de todo programa ejecutable en Go.
//
// REGLA FUNDAMENTAL: todo ejecutable Go tiene exactamente UN paquete "main"
// con exactamente UNA función "main()". Es el equivalente de:
//   - src/main.ts en NestJS  (el bootstrap)
//   - index.js / server.js en Express
//
// DIFERENCIA CLAVE CON NODE.JS:
// Node.js interpreta el código JS en runtime. Go COMPILA todo el programa
// (este archivo + todos sus imports) a un binario nativo del sistema operativo.
// En producción no hay runtime de Go — solo el binario resultante.
// Esto tiene implicaciones importantes para el Docker (ver Dockerfile).
package main

import (
	// "log" provee logging con timestamp automático.
	// Principales funciones:
	//   log.Println(...)  → imprime con timestamp y sigue la ejecución
	//   log.Printf(...)   → como Println pero con formato (verbos como fmt.Sprintf)
	//   log.Fatal(...)    → imprime con timestamp y llama os.Exit(1)
	//   log.Panic(...)    → imprime y hace panic
	// En NestJS usarías Logger de @nestjs/common, Winston, o Pino.
	"log"

	// "net/http" es el servidor HTTP incluido en la biblioteca estándar de Go.
	// A diferencia de Node.js donde necesitas Express, Fastify o Hapi para
	// tener un servidor HTTP usable, Go incluye uno production-ready en stdlib.
	// http.ListenAndServe es todo lo que necesitas para servir HTTP.
	"net/http"

	// chi es un router ligero compatible con net/http.
	// Solo aporta lo que net/http no tiene: parámetros de ruta ({id}),
	// grupos de rutas, y una API de middleware limpia.
	// En el ecosistema de Node sería Express o Fastify — pero chi es
	// MUCHO más minimalista. No es un framework completo, solo un router.
	"github.com/go-chi/chi/v5"

	// chi/middleware provee middlewares pre-construidos listos para usar.
	// middleware.Logger   → loguea cada request (método, path, status, latencia)
	// middleware.Recoverer → captura panics y devuelve 500 en vez de crashear
	"github.com/go-chi/chi/v5/middleware"

	// Nuestros paquetes internos. El prefijo es el module name de go.mod.
	// En go.mod: module tasks-api
	// Por eso el import es "tasks-api/handler" y "tasks-api/store".
	// En TypeScript sería: import { TaskHandler } from './handler/task'
	// pero en Go importas el PAQUETE completo y accedes como handler.Xxx.
	"tasks-api/handler"
	"tasks-api/store"
)

// main es el punto de entrada del programa.
// Go la llama automáticamente al ejecutar el binario.
// No recibe argumentos (los args se leen con os.Args si los necesitas)
// y no devuelve valores (los errores fatales se reportan con log.Fatal u os.Exit).
func main() {
	// INYECCIÓN DE DEPENDENCIAS MANUAL — el patrón Go para DI.
	//
	// En NestJS, el framework instancia y conecta los providers automáticamente
	// gracias al sistema de módulos y decoradores. Tú declaras las dependencias
	// en el constructor y NestJS las resuelve al arrancar.
	//
	// En Go no hay IoC container. Creamos las dependencias nosotros mismos
	// y las "inyectamos" pasándolas como argumentos. Es más verboso pero
	// completamente explícito — nada sucede por magia.
	//
	// Orden: primero el store (sin dependencias), luego el handler (necesita el store).
	s := store.NewTaskStore()    // "repositorio" en memoria
	h := handler.NewTaskHandler(s) // handler que depende del store

	// chi.NewRouter() crea el router principal (el mux).
	// Equivale a:
	//   Express: const app = express()
	//   NestJS:  const app = await NestFactory.create(AppModule)
	r := chi.NewRouter()

	// r.Use() registra middlewares globales que se ejecutan en CADA request,
	// antes de llegar al handler correspondiente. Se aplican en orden de registro.
	//
	// En NestJS: app.use(middleware) o @UseInterceptors() / @UseGuards()
	// En Express: app.use(middleware)
	//
	// middleware.Logger:
	//   Loguea cada request con método, path, status y duración.
	//   Output: "GET /tasks 200 1.234ms"
	//
	// middleware.Recoverer:
	//   Captura panics dentro de los handlers y responde con 500.
	//   En Go, un "panic" es como una excepción no capturada en JS.
	//   Sin Recoverer, un panic mataría el servidor entero.
	//   Con Recoverer, solo falla esa request específica.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Registro explícito de rutas: Method + Path + Handler.
	//
	// chi usa {param} para parámetros de ruta (a diferencia de :param en Express).
	// Cada handler es un método del struct TaskHandler.
	//
	// En NestJS esto es implícito vía decoradores en el controller:
	//   @Get()            ↔  r.Get("/tasks", h.GetAll)
	//   @Post()           ↔  r.Post("/tasks", h.Create)
	//   @Get(':id')       ↔  r.Get("/tasks/{id}", h.GetByID)
	//   @Put(':id')       ↔  r.Put("/tasks/{id}", h.Update)
	//   @Delete(':id')    ↔  r.Delete("/tasks/{id}", h.Delete)
	//
	// En Go es explícito: ves exactamente qué método va a qué handler.
	// Ventaja: el flujo es trazable de arriba a abajo sin "magia" de decoradores.
	r.Get("/tasks", h.GetAll)
	r.Post("/tasks", h.Create)
	r.Get("/tasks/{id}", h.GetByID)
	r.Put("/tasks/{id}", h.Update)
	r.Delete("/tasks/{id}", h.Delete)

	log.Println("Server running on :8080")

	// http.ListenAndServe(":8080", r) inicia el servidor HTTP y BLOQUEA
	// el goroutine principal esperando conexiones entrantes indefinidamente.
	// Solo retorna si ocurre un error fatal (puerto ocupado, permisos, etc.).
	//
	// Internamente, Go crea un nuevo goroutine para CADA request entrante.
	// Así maneja concurrencia: no con un event loop como Node.js, sino con
	// goroutines (hilos ultra-ligeros del runtime de Go, ~2KB de stack).
	// Miles de goroutines pueden correr simultáneamente sin el overhead de threads OS.
	//
	// log.Fatal envuelve la llamada:
	//   si ListenAndServe devuelve un error → lo loguea y llama os.Exit(1)
	//   (ListenAndServe nunca retorna nil — siempre es un error si retorna)
	//
	// Equivalente en NestJS:
	//   await app.listen(3000);  ← async porque Node usa event loop
	// En Go es síncrono y bloqueante — el main goroutine se queda aquí.
	log.Fatal(http.ListenAndServe(":8080", r))
}
