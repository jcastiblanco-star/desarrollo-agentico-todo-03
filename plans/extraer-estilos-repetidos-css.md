# Chore: Extraer los estilos repetidos en clases CSS reutilizables

## Encargo y resultado esperado

Extraer los estilos repetidos de `public/assets/app.css` en clases CSS reutilizables (bases compartidas y valores comunes en variables) y aplicarlas, sin cambiar el aspecto visual ni el comportamiento de la aplicación.

Resultado esperado:

- La hoja de estilos declara una vez cada grupo de propiedades que hoy se repite en varios selectores.
- Los colores, radios y tamaños que se repiten con el mismo valor pasan a variables en `:root`.
- Para cualquier elemento, la estilo computado antes y después del cambio es idéntica en todos los estados comprobados (reposo, hover, foco, completada, error, móvil, escritorio).
- Los nombres de clase que usan el HTML, `app.js` y los tests Go no cambian.

## Estado actual y restricciones

- Rama `main`, adelantada 1 commit a `origin/main`. Hay cambios **sin commitear** de la funcionalidad de reordenar (`plans/reordenar-tareas-drag-and-drop.md`) que tocan exactamente los mismos archivos que este chore: `public/assets/app.css` (líneas 395-428 y 455-458 añadidas), `public/assets/app.js`, `templates/tasks/index.plush.html` y `templates/application.plush.html` (`?v=4`). Este chore **no debe mezclarse** con esos cambios (ver Riesgos).
- Hojas de estilo servidas desde `public/` mediante `public/embed.go` y `actions/app.go:60`. No hay bundler, ni `package.json`, ni preprocesador: el CSS se escribe a mano.
- `templates/application.plush.html:7` carga `/assets/app.css?v=4`. Cualquier cambio de `app.css` debe ir acompañado de subir ese parámetro para evitar caché en navegadores que ya tengan la hoja.
- `app.js` depende de estos nombres de clase, que **no deben renombrarse**: `task`, `task-completed`, `task-status`, `task-toggle-button`, `task-title`, `task-actions`, `task-handle`, `task-dragging`, `task-list`, `toggle-label`, `field-error` (vía `hidden`), y de los atributos `data-task-id`, `data-task-toggle`.
- `actions/tasks_test.go:355` comprueba la cadena exacta `class="icon-action task-handle"`. Por eso el HTML no se toca en este chore (ver decisión 2).
- Orden de cascada significativo (tres casos verificados en el archivo):
  - `.task-toggle-button, .icon-action` (línea 140) va antes de `.task-handle` (línea 395), que sobrescribe `cursor` a `grab`.
  - `.button` (273) va antes de `.button-primary` (297) y de sus `:hover` (293, 303).
  - Los bloques `@media` (429, 484, 502) deben conservar sus reglas en el mismo orden relativo.
- Todas las reglas usan selectores de una sola clase o de tipo; agrupar selectores no cambia la especificidad.

## Decisiones tomadas

1. **Alcance solo CSS.** No se modifican plantillas, `app.js`, Go ni tests. Razón: el HTML ya usa clases semánticas que sirven de hook para JS y para tests; añadir clases utilitarias en el marcado obligaría a tocar `tasks_test.go` y a coordinar con el cambio de arrastrar y soltar.
2. **Reutilización por selectores agrupados + variables**, no por clases utilitarias nuevas en el HTML. Las "clases reutilizables" son bases compartidas (`.icon-button`-style: un selector agrupado que hereda el mismo bloque de propiedades).
3. **Variables solo para valores repetidos con el mismo literal** (≥2 usos iguales). No se crean variables para valores que aparecen una sola vez.
4. **Mismo orden de bloques en el archivo.** Se refactoriza en sitio; no se reordena la hoja.

## Alcance

### Incluido

- `public/assets/app.css`:
  - Nuevas variables en `:root` para colores, radio, tamaños de control y tamaños de texto repetidos.
  - Sustitución de literales repetidos por `var(...)` con el mismo valor.
  - Agrupación de selectores que comparten el mismo bloque de propiedades (ver lista en "Implementación").
- `templates/application.plush.html`: solo el parámetro de caché `?v=4` → `?v=5` si cambia `app.css` (y `app.js` solo si se modifica, lo cual no ocurre aquí).

### Excluido

- Cualquier cambio de valor visual (colores, tamaños, espaciados, transiciones, breakpoints).
- Renombrar clases, añadir clases al HTML, o tocar `templates/`, `actions/`, `models/`, `migrations/`, `fixtures/`.
- Cambios en `app.js`, en la lógica de arrastrar y soltar o en `plans/reordenar-tareas-drag-and-drop.md`.
- Reescribir la hoja en otro formato (Sass, PostCSS, utilidades tipo Tailwind) o añadir dependencias.
- Commit o push. Lo gestiona `/git:commit` o `/commit-otros` al final.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.css` | Único archivo de estilos. Todo el trabajo ocurre aquí. |
| `templates/application.plush.html:7` | Parámetro `?v=` de `app.css` (subir si cambia la hoja). |
| `templates/tasks/index.plush.html`, `templates/tasks/edit.plush.html`, `templates/_flash.plush.html` | Solo de lectura: confirmar qué clases se usan realmente. |
| `public/assets/app.js` | Solo de lectura: clases y atributos que no pueden cambiar (líneas 14-29, 47-68, 112-129, 173, 186-191). |
| `actions/tasks_test.go:354-355` | Solo de lectura: cadena de clase que no puede cambiar. |
| `public/embed.go`, `actions/app.go:60` | Solo de lectura: confirman que la hoja se sirve tal cual desde `public/`. |

## Repeticiones identificadas en `app.css` (líneas del archivo actual)

Propiedades compartidas entre selectores:

- **Control cuadrado de 44 px** (`min-height`/`width`/`height: 44px`): 55, 74, 148, 240-241, 251, 278, 354. Móvil 36 px: 459, 463, 480.
- **Radio 6px**: 76, 154, 257, 281, 324, 340.
- **Centrado en flex** (`inline-flex` + `align-items:center` + `justify-content:center`): 143-145 (`.task-toggle-button, .icon-action`), 275-276 (`.button`); `align-items:center` en 51-53, 71-73, 202-203, 351-352.
- **Centrado en grid** (`display:grid; place-items:center`): 160-161 (`.completion-icon`), 237-238 (`.add-mark`).
- **Botón sin estilo** (`font: inherit`, `background: transparent`, `border: 0`, `cursor: pointer`, `text-decoration: none`): 150-156 y 290 (`.button` tiene `cursor: pointer` también).
- **Bloque de aviso** (`margin`, `padding`, `border-radius: 6px`, `font-size: 0.85rem`): `.field-error` (321-328) y `.flash` (336-342) tienen la misma base; difieren en margen y colores.
- **Color muted** `var(--color-muted)`: 114, 187, 265, 317, 332, 357. (Algunas son placeholder o texto; el valor es idéntico.)
- **Tamaños de texto**: `0.85rem` en 285, 327, 341, 356, 368; `0.8rem` en 115, 318; `0.9rem` en 79, 333.
- **Colores de hover/fondo literales**:
  - `#f7e9e4`: 77 (`.nav-link`) y 177 (`.completion-icon` hover).
  - `#f2ded6`: 85.
  - `#f2f2ee`: 218.
  - `#fcf0ec`: 223.
  - `#eaeae4`: 294; `#f6f6f2`: 283.
  - `#fff0eb`: 326; `#912a20`: 325.
  - `#858580` / `#777771` / `#96968f` / `#92928a` / `#dadad4`: uso único en su selector; no se extraen (quedan como están).
- **Hover con el mismo patrón**: `.nav-link:hover` (84), `.icon-action:hover` (216), `.button:hover` (293), `.back-link:hover` (361). Se mantienen separados si el color de hover difiere.

## Implementación paso a paso

Prerrequisito: el cambio de arrastrar y soltar debe estar commiteado o aparcado (ver Riesgos). Si no lo está, no empezar.

1. **Capturar la línea base (antes de tocar nada).** Seguir "Tests y validación" → paso A. Guardar el resultado en la carpeta de scratchpad, no en el repositorio.
2. **Añadir variables en `:root`** (`app.css:17-27`). Mantener las existentes y añadir:
   - `--color-primary-soft: #f7e9e4;`
   - `--color-primary-soft-hover: #f2ded6;`
   - `--color-hover: #f2f2ee;`
   - `--color-danger-soft: #fcf0ec;`
   - `--color-error-text: #912a20;`
   - `--color-error-bg: #fff0eb;`
   - `--color-button-bg: #f6f6f2;`, `--color-button-hover: #eaeae4;`, `--color-button-border: #dadad4;`
   - `--radius: 6px;`
   - `--control-size: 44px;`
   - `--text-small: 0.85rem;`

   Cada valor debe ser el literal exacto de la tabla anterior; no redondear ni normalizar.
3. **Sustituir literales por variables** en los selectores de la lista de la sección anterior (`#f7e9e4` → `var(--color-primary-soft)`, etc.). Hacer una sustitución por grupo de color y comprobar el diff tras cada grupo.
4. **Agrupar selectores con bloques idénticos** sin mover las reglas de sitio:
   - `.button` ya es la base de `.button-primary`; no tocar.
   - Crear una base de aviso compartida: `.field-error, .flash { padding: 0.7rem 0.85rem; border-radius: var(--radius); font-size: var(--text-small); }` en la posición de `.field-error`. Quitar esas tres propiedades de `.field-error` y `.flash`, dejando en cada uno solo margen, color y fondo. **Verificar** que `.flash` mantiene `padding: 0.75rem 0.9rem` (no `0.7rem 0.85rem`): si difieren, no se agrupan o se conserva el valor propio en su selector (en la hoja actual `.flash` usa `0.75rem 0.9rem`).
   - Agrupar la base de botón sin estilo si el bloque es idéntico en `.icon-action` y `.task-toggle-button` (ya agrupados; no duplicar `cursor`).
5. **No reordenar ni mover** reglas entre `@media`. No cambiar `:hover` ni `:focus-visible`.
6. **Subir la caché:** en `templates/application.plush.html:7` cambiar `app.css?v=4` → `app.css?v=5`. Dejar `app.js?v=4` como está (no cambia `app.js`). *Nota*: si el cambio de arrastrar y soltar ya subió a `v=4` y el commit está hecho, esta subida pasa a `v=5`; si ya se subió a `v=5` en el cambio previo, no volver a subir.
7. **Revisión manual del diff:** `git diff -- public/assets/app.css templates/application.plush.html` debe mostrar solo sustituciones de valor por `var()`, adición de variables y agrupaciones. Ninguna propiedad nueva, ninguna eliminada sin equivalente.
8. Ejecutar la validación posterior (paso B y C). Si la comparación de estilos computados no es idéntica, deshacer el último grupo y repetir.

## Tests y validación

Antes y después del cambio (mismo script, mismo entorno):

**A. Línea base y comparación de estilos computados (prueba principal).**
- Levantar la app con la skill `/env:start` (o `/run`); no inventar comandos. Confirmar que responde en `http://localhost:3000/`.
- Con la skill `playwright-cli` (o Playwright vía script en el scratchpad), recorrer:
  - `/` con lista vacía (requiere tener la BD sin tareas o usar una copia temporal de fixtures).
  - `/` con tareas pendientes y completadas (las del fixture).
  - `/` con el error de título vacío (enviar el formulario de alta vacío).
  - `/tasks/<id>/edit` y su error de validación.
- Anchos de viewport: 1280×800, 760 (límite de `@media (max-width: 760px)`), 761, 390, 360 (límite de `max-width: 380px`).
- Estados: reposo; hover sobre `.nav-link`, `.task-toggle-button`, `.icon-action`, `.button`, `.back-link`; foco con teclado (`:focus-visible`); `task-completed`.
- Para cada elemento seleccionado (`document.querySelectorAll('*')`), volcar `getComputedStyle` de todas las propiedades a JSON. Guardar `before.json` y `after.json` en el scratchpad y comparar con `diff`. **Resultado esperado: diferencia vacía.**
- Captura de pantalla de cada combinación antes y después; comparar los PNG (cualquier diferencia de píxeles es fallo).

**B. Comportamiento (no debe cambiar).**
- Completar una tarea con el botón circular: el estilo `task-completed` se aplica y se conserva al recargar.
- Editar, eliminar (con confirmación) y crear tarea: sin cambios.
- Arrastrar y soltar y `Alt+↑/↓`: solo si la funcionalidad de reordenar ya está integrada; si no, omitir.
- Comprobar que `app.js` no lanza errores en la consola del navegador.

**C. Comprobaciones del repositorio (CI).** El cambio es solo CSS y una línea de plantilla, pero CI se ejecuta igual:
- `gofmt -l .` debe devolver vacío (no hay Go nuevo; falla solo si el árbol ya lo incumple).
- `go vet ./...`
- `go build ./...`
- `go test -p 1 ./...` requiere PostgreSQL con la base `todo_test` y las migraciones (`soda migrate -e test`, como en `.github/workflows/ci.yaml`). Si el entorno local no tiene Postgres, anotar que no se ha ejecutado; no omitir en silencio.
- `docker build -t todo .` solo si se quiere confirmar que la imagen embebe la hoja nueva (`public/embed.go`).

**D. Verificación estática.** `grep -n "#f7e9e4\|#f2ded6\|#f2f2ee\|#fcf0ec\|#fff0eb\|#912a20\|#eaeae4\|#f6f6f2\|#dadad4" public/assets/app.css` debe devolver solo las líneas de definición en `:root`.

## Riesgos y recuperación

- **Conflicto con el cambio de arrastrar y soltar (alto).** `app.css`, `app.js`, `index.plush.html` y `application.plush.html` tienen cambios sin commitear de otra funcionalidad. Mezclarlos ensucia el diff y puede romper `?v=`. Mitigación: empezar solo cuando ese cambio esté commiteado en su propio commit; si no, trabajar en una rama nueva desde el estado actual y no tocar esos bloques hasta resolverlo.
- **Caché del navegador (medio).** Sin subir `?v=`, un navegador con la hoja antigua verá estilos mezclados. Mitigación: paso 6.
- **Cambio de cascada por agrupar (medio).** Mover una propiedad de `.flash` a una base compartida puede cambiar el resultado si el valor propio difiere. Mitigación: los valores propios se conservan en su selector y se verifican con la comparación A.
- **Precisión de variables (bajo).** `var()` con el mismo literal produce el mismo valor computado; el riesgo aparece solo si se escribe mal un valor. Mitigación: la tabla y la comparación A.
- **Recuperación.** Todo el cambio está en `public/assets/app.css` y una línea de `application.plush.html`. Revertir con `git checkout -- public/assets/app.css templates/application.plush.html` **solo** si el cambio de arrastrar y soltar ya está commiteado; si no, revertir a mano las líneas del chore (no usar `checkout` porque borraría el trabajo sin commitear de la otra funcionalidad).
- **Sin efectos operativos.** No hay migraciones, datos ni dependencias nuevas. No hay despliegue que coordinar más allá de la imagen Docker.

## Incertidumbres pendientes

- Si el cambio de arrastrar y soltar se commitea antes o después de este chore determina el número de `?v=`.
- No se ha ejecutado la aplicación ni la comparación de estilos durante la planificación: los números de línea provienen de la lectura del archivo tal como está en el árbol de trabajo actual.
