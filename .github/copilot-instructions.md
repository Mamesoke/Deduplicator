# Instrucciones para Copilot

## Proyecto

Deduplicator es una herramienta CLI escrita en Go para localizar archivos duplicados por contenido. El módulo está declarado como `deduplicator` en `go.mod` y requiere Go 1.24.2 o posterior.

## Estructura

- `main.go`: define las opciones CLI, valida el formato y el algoritmo de hash, coordina el escaneo, la salida y el borrado.
- `deduplicator/`: lógica reutilizable del programa:
  - `walker.go`: recorrido de directorios, exclusiones, workers y caché.
  - `hasher.go`: implementaciones de SHA-256, SHA-1, SHA-512 y MD5.
  - `deduplicator.go`: agrupación de archivos con el mismo hash.
  - `formatter.go`: salida legible y reporte JSON.
  - `deleter.go`: simulación y borrado de copias redundantes.
  - `cache.go`, `types.go`, `config.go`: persistencia de caché, tipos y configuración compartida.
- Las pruebas están junto al código correspondiente en archivos `*_test.go`.

## Desarrollo y validación

- Ejecuta las pruebas con `go test ./...`.
- Compila todos los paquetes con `go build ./...`.
- Ejecuta desde el repositorio con `go run . -dir="<directorio>"`; consulta `README.md` para las opciones CLI y sus valores admitidos.
- El proyecto no declara dependencias de terceros en `go.mod`; prefiere la biblioteca estándar salvo que una necesidad justifique añadir una dependencia.
- Las pruebas usan el paquete estándar `testing` y crean datos temporales con `t.TempDir()`.

## Comportamiento que hay que preservar

- El escaneo puede devolver archivos junto con errores parciales. No descartes ni ocultes esos errores; la CLI los informa y termina con un código distinto de cero.
- En formato JSON, el reporte va a `stdout` y los mensajes de diagnóstico a `stderr`; conserva `stdout` como JSON válido para poder canalizarlo a otras herramientas.
- Las exclusiones se comparan con el nombre de cada archivo o directorio mediante patrones de `filepath.Match`, no con la ruta completa. Las exclusiones predeterminadas se definen en `main.go`.
- La caché `.dedupcache.json` se crea en el directorio escaneado; valida el algoritmo, tamaño y tiempo de modificación antes de reutilizar un hash. No la trates como código fuente ni la incluyas en cambios funcionales.
- `-delete` elimina archivos de forma permanente y no pide confirmación. Mantén la comparación byte a byte antes de borrar una copia y conserva `-dry-run` como opción segura para previsualizar las rutas.

## Convenciones de implementación

- Mantén la separación entre la CLI en `main.go` y la lógica del paquete `deduplicator`.
- Propaga errores con contexto; no los ignores ni los conviertas en resultados aparentemente exitosos.
- Al cambiar un comportamiento visible, actualiza las pruebas relacionadas y la documentación correspondiente en `README.md`.
