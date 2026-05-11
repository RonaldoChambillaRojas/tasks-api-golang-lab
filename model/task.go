// Package model contiene las estructuras de datos (entidades) del dominio.
// En NestJS usarías clases con decoradores @Entity() para TypeORM,
// o simplemente interfaces/clases para DTOs. En Go, usamos "structs".
//
// Un "package" en Go es la unidad de modularización. Todo archivo .go
// dentro de la misma carpeta pertenece al mismo paquete. Es como un módulo
// de Node, pero definido por directorio, no por archivo.
package model

// Task representa una tarea en el sistema.
//
// En TypeScript/NestJS escribirías algo como:
//
//	export class Task {
//	  id: string;
//	  title: string;
//	  done: boolean;
//	}
//
// En Go no existen clases. En cambio usamos "struct", que es un tipo de dato
// compuesto: un conjunto de campos tipados agrupados bajo un nombre.
// Los structs NO tienen métodos por defecto — los métodos se agregan aparte
// usando "receivers" (lo verás en store/task.go).
//
// REGLA DE ORO en Go — visibilidad por mayúscula inicial:
//
//	Campo/función con MAYÚSCULA → EXPORTADO (equivale a "public")
//	Campo/función con minúscula → NO EXPORTADO (equivale a "private")
//
// Esto aplica a structs, funciones, variables, constantes — absolutamente todo.
// No existe la keyword "public" o "private": la primera letra lo decide.
type Task struct {
	// ID es el identificador único de la tarea.
	//
	// El `json:"id"` se llama "struct tag": es metadato en forma de string
	// que Go lee en tiempo de ejecución usando reflection. Le dice al
	// encoder/decoder JSON que este campo debe ser "id" (minúscula) en el
	// JSON de entrada/salida, aunque el campo Go se llame "ID" (mayúscula).
	//
	// En NestJS usarías el decorador @ApiProperty() para Swagger, o el
	// decorador @Expose() con class-transformer. Las struct tags son el
	// equivalente en Go, pero son solo strings — no ejecutan código como
	// los decoradores de TypeScript.
	//
	// Formato: `nombreLib:"clave,opciones"`. Opciones comunes de json:
	//   json:"nombre"          → usa ese nombre en el JSON
	//   json:"nombre,omitempty"→ omite el campo si es zero value
	//   json:"-"               → siempre ignora este campo
	ID string `json:"id"`

	// Title es el texto descriptivo de la tarea.
	// "string" en Go es igual que en TypeScript: cadena de texto inmutable.
	// En Go todos los strings son UTF-8 por defecto.
	Title string `json:"title"`

	// Done indica si la tarea fue completada.
	// "bool" en Go equivale a "boolean" en TypeScript.
	//
	// ZERO VALUES — concepto clave de Go:
	// Cada tipo tiene un "zero value" (valor por defecto cuando se declara
	// sin inicializar). No existe "undefined" como en JS.
	//   string  → ""    (cadena vacía)
	//   int     → 0
	//   bool    → false
	//   pointer → nil
	//   slice   → nil
	//   map     → nil
	// Esto significa que `var t Task` crea {ID:"", Title:"", Done:false}
	// en lugar de undefined o null.
	Done bool `json:"done"`
}
