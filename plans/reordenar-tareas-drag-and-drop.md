# Feature: Reordenar las tareas con arrastrar y soltar

## Encargo y criterios de aceptación

### Requisitos explícitos (del encargo)

- Añadir arrastrar y soltar (drag and drop) a la lista de tareas para poder reordenarla.

### Criterios de aceptación

1. Cada tarea de la lista tiene un asidero (handle) que permite arrastrarla a otra posición con el ratón y con el dedo en pantallas táctiles.
2. Al soltar la tarea, la nueva ordenación se guarda en PostgreSQL y se mantiene al recargar la página.
3. El orden de todas las tareas (pendientes y completadas) se respeta tras recargar. Las tareas nuevas aparecen al final.
4. Si el guardado falla o la lista ha cambiado en otra pestaña, la lista no queda en un estado engañoso: se restaura el orden anterior o se recarga, y el usuario ve un aviso.
5. La reordenación se puede hacer también con teclado (inferencia, ver más abajo).
6. Completar, editar, eliminar y crear tareas siguen funcionando igual.

### Inferencias (no pedidas explícitamente; pendientes de confirmar)

- **Persistencia del orden:** el orden actual depende de `created_at, id` (`actions/tasks.go`, `TasksIndex`). Hace falta una columna nueva; no existe ningún campo de posición.
- **Teclado:** un asidero solo con ratón deja fuera a usuarios de teclado. Se propone mover la tarea enfocada con `Alt+↑` / `Alt+↓`.
- **Móvil:** la app tiene diseño para ≤760 px (`public/assets/app.css:407`). La API nativa de arrastrar y soltar de HTML5 no funciona con el tacto, así que se usan Pointer Events.
- **Sin dependencias nuevas:** el repositorio no tiene `package.json` ni bundler; el JavaScript vive en `public/assets/app.js` y se carga con `defer`.

## Comportamiento actual y propuesta

### Actual

- `templates/tasks/index.plush.html` pinta las tareas en `<ul class="task-list">` en el orden que devuelve `TasksIndex`: `tx.Order("created_at asc, id asc")` (`actions/tasks.go:21`).
- No hay campo de orden en `models.Task` (`models/task.go:12-17`) ni en la tabla `tasks` (`migrations/20260901000000_create_tasks.up.fizz`).
- `TasksCreate` también ordena por `created_at, id` cuando vuelve a pintar la lista con errores (`actions/tasks.go:45`).
- `app.js` ya tiene un patrón para formularios con `data-task-toggle` que actualiza el DOM antes de la petición (`public/assets/app.js:12-34`). Ese código tiene un fallo conocido (ver *Riesgos*).

### Propuesta

1. **Persistencia:** nueva columna `position integer NOT NULL DEFAULT 0` en `tasks`, rellenada con la posición actual (`row_number()` sobre `created_at, id`) en la misma migración.
2. **Orden:** todas las consultas de listado ordenan por `position asc, created_at asc, id asc`. El desempate por `created_at, id` cubre filas con posiciones repetidas.
3. **Alta:** una tarea nueva recibe `position = MAX(position) + 1`, es decir, queda al final.
4. **Reordenación:** el cliente envía la lista completa de identificadores en el nuevo orden a `POST /tasks/reorder` con cabecera `X-CSRF-Token`. El servidor comprueba que el conjunto de identificadores coincide exactamente con las tareas existentes y reescribe `position` en `0..n-1` dentro de la transacción de la petición. Responde `204 No Content`.
5. **Cliente:** el asidero es un `<button type="button">` con `aria-label` dentro de `.task-actions`. El arrastre usa Pointer Events (`pointerdown` con `setPointerCapture`, `pointermove`, `pointerup`/`pointercancel`). Al soltar, si el orden ha cambiado, se envía la petición. Con teclado, `Alt+↑/↓` intercambia la tarea con su vecina y envía la misma petición.
6. **Fallos:** si la respuesta no es 204 se restaura el orden original en el DOM y se muestra un aviso en `role="alert"`; un 409 (la lista cambió en otra pestaña) recarga la página.

### Por qué esta forma

- **Lista completa en lugar de "mover una tarea de A a B":** es idempotente, evita que dos pestañas dejen posiciones duplicadas y no necesita lógica de desplazamiento en el servidor.
- **`POST /tasks/reorder` en lugar de `PUT /tasks/order`:** el router ya tiene `PUT /tasks/{task_id}` (`actions/app.go:55`). Un `PUT /tasks/order` dependería del orden de registro para no ser capturado como `task_id`. Con `POST` no hay colisión.
- **Pointer Events en lugar de HTML5 DnD:** funciona con ratón, lápiz y tacto con un solo código.
- **Posiciones densas `0..n-1`:** simplifican la validación de la petición y mantienen `position` coherente tras cada reordenación. Las eliminaciones dejan huecos que no afectan al orden.

## Alcance

### Incluido

- Migración que añade `position` y la rellena con el orden actual.
- Campo `Position` en `models.Task`, orden por posición en listado y en el re-render de errores de creación.
- Asignación de posición al final en `TasksCreate`.
- Ruta y acción `POST /tasks/reorder` con validación de conjunto completo.
- Asidero visual y accesible en cada tarea, con arrastre por ratón/tacto y `Alt+↑/↓`.
- Estilos del asidero, del estado "arrastrando" y del aviso de error, incluidos los puntos de ruptura existentes.
- Actualización del parámetro de caché `?v=3` → `?v=4` en `application.plush.html` para `app.css` y `app.js`.

### Excluido

- Ordenación por otros criterios, filtros o búsqueda (no existen en la app).
- Arrastrar entre listas o agrupar tareas.
- Modo sin JavaScript: la reordenación requiere JS; el resto de la lista sigue siendo utilizable.
- Animaciones complejas (se usa una transición simple, que respeta `prefers-reduced-motion`).
- Cambios en la lógica de completar tareas. El fallo de `fetch` de `data-task-toggle` tiene su propio plan (`plans/completar-tarea-no-persiste.md`).
- Nuevas dependencias de JavaScript o de Go.

## Diseño por capas

### Persistencia y migraciones

Nueva migración con par `up`/`down`, en formato SQL porque necesita un `UPDATE` de backfill (el `embed` de `migrations/embed.go` ya incluye `*.sql`):

- `migrations/20261019000000_add_position_to_tasks.up.sql`:

  ```sql
  ALTER TABLE tasks ADD COLUMN position integer NOT NULL DEFAULT 0;

  UPDATE tasks AS t
  SET position = o.pos
  FROM (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC, id ASC) - 1 AS pos
    FROM tasks
  ) AS o
  WHERE t.id = o.id;

  CREATE INDEX tasks_position_idx ON tasks (position);
  ```

- `migrations/20261019000000_add_position_to_tasks.down.sql`:

  ```sql
  DROP INDEX IF EXISTS tasks_position_idx;
  ALTER TABLE tasks DROP COLUMN position;
  ```

El índice es opcional con pocas filas; se incluye por coherencia con el orden de listado. Si se prefiere mínimo, se puede omitir.

La migración de semilla `20260901000001_seed_tasks` se ejecuta antes, así que sus filas también reciben posición.

### Modelo (`models/task.go`)

- Añadir `Position int \`json:"position" db:"position" form:"-"\`` a `Task`. `form:"-"` evita que el formulario de edición o de creación lo sobrescriba, igual que `Completed`.
- No añadir validación nueva; el conjunto se valida en la acción porque depende de las demás filas.
- Opcional: una función `func NextPosition(tx *pop.Connection) (int, error)` con `COALESCE(MAX(position), -1) + 1`, usada por `TasksCreate`. Mantenerla en `models` evita SQL en la acción.

### Acción y ruta (`actions/tasks.go`, `actions/app.go`)

- `TasksIndex`: `tx.Order("position asc, created_at asc, id asc")`.
- `TasksCreate`: asignar `task.Position` antes de `ValidateAndCreate`; cambiar también el orden del re-render de errores.
- Nueva acción `TasksReorder`:
  1. Enlazar el cuerpo JSON `{"ids": ["…"]}` a `struct{ IDs []uuid.UUID \`json:"ids"\` }`. Error de formato → `422`.
  2. Cargar todas las tareas (`tx.All(&tasks)`) y comparar: mismo número de elementos, sin duplicados en `ids`, y todos los `ids` existentes. Si no coincide → `409 Conflict` (la lista cambió en otra pestaña). Si `ids` está vacío y no hay tareas, responder `204` sin cambios.
  3. Por cada posición `i`, ejecutar `UPDATE tasks SET position = ? WHERE id = ?` mediante `tx.RawQuery(...).Exec()`. Se usa SQL directo para no tocar `updated_at` ni revalidar el título de todas las filas.
  4. Responder `c.Render(http.StatusNoContent, nil)` o `c.Response().WriteHeader(204)`, según lo que permita Buffalo en esta versión; comprobarlo en el código de `actions/` antes de implementar.
- Registrar en `app.go`, junto a las demás rutas de tareas:

  ```go
  app.POST("/tasks/reorder", TasksReorder)
  ```

  La petición pasa por `csrf.New` y `popmw.Transaction`, igual que el resto. La cabecera `X-CSRF-Token` es suficiente para el middleware (ver `plans/completar-tarea-no-persiste.md`, sección de CSRF).

### Renderizado (`templates/tasks/index.plush.html`)

- En cada `<li class="task">` añadir `data-task-id="<%= t.ID %>"`.
- Dentro de `.task-actions`, como primer elemento, el asidero:

  ```html
  <button
    type="button"
    class="icon-action task-handle"
    aria-label="Reordenar tarea: <%= t.Title %>"
    title="Arrastrar para reordenar (o Alt+↑ / Alt+↓)"
  >
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round">
      <path d="M9 6h.01M15 6h.01M9 12h.01M15 12h.01M9 18h.01M15 18h.01" />
    </svg>
  </button>
  ```

  Colocarlo en `.task-actions` evita cambiar la rejilla de `.task` (`public/assets/app.css:122`). En escritorio, `.task-actions` ya aparece al pasar el ratón o al enfocar (`app.css:395-405`); en móvil es siempre visible (`app.css:432`).
- Actualizar `?v=3` → `?v=4` en `application.plush.html` para `app.css` y `app.js`.

### JavaScript (`public/assets/app.js`)

- Mantener el manejador actual de `submit` intacto.
- Nuevo bloque para `.task-handle`:
  - `pointerdown` (solo botón principal o táctil): guardar el orden inicial (`ids` de `.task[data-task-id]`), marcar la tarea como `task-dragging`, `setPointerCapture`.
  - `pointermove`: desplazar la tarea con `transform: translateY(...)` y, cuando su centro cruce el centro de un hermano, moverla en el DOM con `insertBefore`/`after`.
  - `pointerup` / `pointercancel`: quitar la clase, y si `ids` ha cambiado, `fetch("/tasks/reorder", { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": token }, credentials: "same-origin", body: JSON.stringify({ ids }) })`.
  - Respuesta ≠ 204: restaurar el orden original (reinsertar los `li` en el orden guardado) y mostrar aviso. `409`: `window.location.reload()`.
- Teclado: en `keydown` sobre `.task-handle` con `altKey` y `ArrowUp`/`ArrowDown`, mover el `li` con `insertBefore` y enviar la misma petición. Mantener el foco en el asidero de la tarea movida (el `li` se reinserta, así que hay que volver a enfocar su `.task-handle`).
- Token CSRF: leerlo de `input[name="authenticity_token"]` dentro del `li` o de cualquier formulario de la página, igual que hace el código actual. No introducir un `meta` nuevo a menos que se desee tocar `application.plush.html`.
- Aviso de error: añadir un elemento `role="alert"` a la página (por ejemplo, tras `.page-header`) creado desde JS, o reutilizar el sistema de `flash` del servidor recargando. Preferir un elemento estático oculto en la plantilla para no crear nodos a demanda.
- Reglas de tacto: en el asidero poner `touch-action: none` (CSS) para que el navegador no haga scroll al arrastrar.

### Estilos (`public/assets/app.css`)

- `.task-handle { cursor: grab; touch-action: none; }`; `cursor: grabbing` durante el arrastre.
- `.task.task-dragging { position: relative; z-index: 1; background: var(--color-surface, #fff); box-shadow: …; }` usando las variables existentes de `:root` (revisar nombres reales antes de usarlos).
- Respetar `prefers-reduced-motion` (ya existe el bloque en `app.css:476`).
- Revisar la vista móvil (≤760 px y ≤380 px) con el asidero visible y sin desbordes horizontales.

## Archivos relevantes

| Ruta | Responsabilidad | Cambio |
|---|---|---|
| `migrations/20260901000000_create_tasks.up.fizz` | Esquema original de `tasks` | Solo lectura |
| `migrations/20260901000001_seed_tasks.up.sql` | Datos de ejemplo | Solo lectura |
| `migrations/20261019000000_add_position_to_tasks.up.sql` | Columna y backfill | **Nuevo** |
| `migrations/20261019000000_add_position_to_tasks.down.sql` | Reversión | **Nuevo** |
| `migrations/embed.go` | Incrusta `*.sql` y `*.fizz` | Solo lectura (ya cubre `.sql`) |
| `models/task.go` | Modelo y validación | Añadir `Position` (y opcionalmente `NextPosition`) |
| `models/task_test.go` | Tests de modelo (`ModelSuite`) | Tests de posición por defecto y de siguiente posición |
| `actions/tasks.go` | Handlers de tareas | Orden por posición, asignación en alta, `TasksReorder` |
| `actions/app.go` | Registro de rutas | `app.POST("/tasks/reorder", TasksReorder)` |
| `actions/tasks_test.go` | Tests de acción (`ActionSuite`) | Tests de orden, alta y reordenación |
| `fixtures/tasks.toml` | Escenarios de prueba | Añadir escenario con posiciones explícitas |
| `templates/tasks/index.plush.html` | Lista de tareas | `data-task-id`, asidero, aviso de error |
| `templates/application.plush.html` | Layout y carga de assets | Subir `?v=3` → `?v=4` |
| `public/assets/app.js` | Interacción en el cliente | Arrastre con Pointer Events, teclado, `fetch` POST |
| `public/assets/app.css` | Estilos | Asidero, arrastre, aviso, móvil |
| `plans/completar-tarea-no-persiste.md` | Plan del bug de `fetch` en toggle | Solo referencia |

## Implementación paso a paso

1. **Migración.** Crear los dos archivos `20261019000000_add_position_to_tasks.{up,down}.sql` con el contenido de *Persistencia*. Comprobar que el nombre es posterior a `20260901000001`.
2. **Modelo.** Añadir `Position` a `models.Task` con las etiquetas indicadas. Añadir `NextPosition` si se opta por ella.
3. **Fixtures.** Añadir a `fixtures/tasks.toml` un escenario `tareas en orden` con tres filas cuyas `position` sean `2, 0, 1` y `created_at` en orden distinto, para demostrar que el orden no depende de la fecha.
4. **Listado y alta.** En `actions/tasks.go`, cambiar los dos `Order(...)` por `position asc, created_at asc, id asc`. En `TasksCreate`, asignar `task.Position` con la siguiente posición antes de `ValidateAndCreate`.
5. **Reordenación.** Implementar `TasksReorder` y registrar la ruta `POST /tasks/reorder` en `actions/app.go`. Verificar en el código de `actions/` cómo Buffalo responde sin cuerpo (`204`) y usar esa forma.
6. **Plantilla.** Añadir `data-task-id` y el asidero dentro de `.task-actions`. Cambiar `?v=3` a `?v=4` en el layout.
7. **Estilos.** Añadir reglas de `.task-handle` y `.task-dragging`, con `touch-action: none`, y revisar móvil.
8. **JavaScript.** Añadir el bloque de arrastre con Pointer Events, la rama de teclado y el manejo de respuestas. No tocar el manejador de `submit`.
9. **Tests.** Escribir los tests de la sección siguiente.
10. **Validación.** Ejecutar las comprobaciones de la sección siguiente y la verificación manual en navegador.

## Tests y validación

### Tests automáticos (Go)

Se añaden a los archivos existentes; no crear suites nuevas.

**`models/task_test.go` (`ModelSuite`)**

- `Test_Task_Position_DefaultsToZero`: crear una tarea sin posición y comprobar `Position == 0` tras recargarla.
- `Test_NextPosition_AppendsAtEnd` (si existe `NextPosition`): con tres tareas en posiciones `0,1,2`, devuelve `3`; con la tabla vacía, devuelve `0`.

**`actions/tasks_test.go` (`ActionSuite`)**

- `Test_TasksIndex_OrdersByPosition`: cargar el escenario `tareas en orden` y comprobar que el HTML muestra los títulos en el orden de `position` (usar `strings.Index` para comparar posiciones relativas).
- `Test_TasksCreate_AppendsAtEnd`: con una tarea existente en posición `4`, crear otra y comprobar que queda en `5`.
- `Test_TasksReorder_Valid`: con tres tareas, enviar el nuevo orden completo; esperar `204` y comprobar `position` `0..2` en el orden enviado.
- `Test_TasksReorder_MissingTask_Conflict`: enviar dos de tres identificadores → `409`, y ninguna posición cambia.
- `Test_TasksReorder_UnknownTask_Conflict`: incluir un identificador inexistente → `409`.
- `Test_TasksReorder_Duplicate_Invalid`: `ids` con un identificador repetido → `422`.
- `Test_TasksReorder_BadBody_Invalid`: cuerpo que no es JSON válido o con `ids` ausente → `422`.
- `Test_TasksIndex_Rendered_HasHandleAndTaskID`: la página contiene `data-task-id` y `task-handle` para cada tarea.

Nota: en `GO_ENV=test` el CSRF está desactivado (ver `plans/completar-tarea-no-persiste.md`). Por tanto, estos tests **no** comprueban el token; esa parte se valida en el navegador.

### Comprobaciones de formato y compilación

- `gofmt -l .` → no debe listar archivos.
- `go vet ./...`
- `go test -p 1 ./...`
- `go build ./...`

### Condiciones para ejecutar los tests

- PostgreSQL accesible en `127.0.0.1:5432` (o `DB_HOST`), con la base `todo_test` creada y usuario `postgres`/`postgres` según `database.yml`.
- Migraciones de test aplicadas, incluida la nueva, antes de ejecutar `go test`. Ejecutar `buffalo test` o migrar `todo_test` con la herramienta de migraciones de pop.
- Los tests de acción usan `os.DirFS("../fixtures")`; el escenario nuevo debe existir en ese archivo.

**Estado en esta sesión de investigación:** Go 1.27.1 está disponible. `psql` no está en el `PATH` y no se ha comprobado Docker ni la base de datos. No se han ejecutado los tests. Antes de implementar, confirmar con `/env:start` que PostgreSQL responde.

### Verificación manual en navegador

Con la aplicación arrancada (`buffalo dev`, `http://localhost:3000`), usando la semilla de tres tareas:

1. **Ratón:** arrastrar «Renovar el dominio de la web» al principio. Recargar: debe seguir arriba.
2. **Persistencia en BD:** `SELECT title, position FROM tasks ORDER BY position;` refleja el orden visual.
3. **Teclado:** enfocar el asidero de una tarea y pulsar `Alt+↑`. La tarea sube, el foco sigue en su asidero, y tras recargar el orden se mantiene.
4. **Táctil:** emulación de móvil (≤380 px y 760 px) en DevTools con modo táctil; arrastrar sin que la página haga scroll.
5. **Fallo de red o 409:** simular con DevTools (modo offline o bloqueo de `/tasks/reorder`) y comprobar que el orden se restaura y aparece el aviso. Para el 409, reordenar en una pestaña, crear una tarea en otra y reordenar en la primera.
6. **Completar y editar siguen funcionando** tras reordenar. (Completar depende del bug previo; ver riesgos.)
7. **Alta:** crear una tarea y comprobar que aparece al final.
8. **Migración con datos:** en una copia de la base con datos reales, aplicar la migración y comprobar que las posiciones reproducen el orden de `created_at`. Bajar la migración (`down`) y volver a subirla.

Usar la skill `playwright-cli` para automatizar los pasos 1, 3 y 5 si se desea repetirlos.

## Riesgos y recuperación

- **Colisión con el bug de toggle (riesgo alto de confusión).** `public/assets/app.js:34` hace `fetch(form.action)` sin método, lo que es un GET que devuelve 404 (`plans/completar-tarea-no-persiste.md`). El nuevo `fetch` de reordenación debe usar `method: "POST"` y la cabecera de token; no copiar el patrón del toggle. Ambos cambios tocan `app.js`: si se implementan en paralelo, hacer `git pull`/rebase con cuidado. Recomendación: corregir el bug de toggle primero o en el mismo lote, para que la verificación manual sea fiable.
- **Doble fuente de verdad (DOM vs servidor).** Si el DOM se reordena y la petición falla, puede quedar desincronizado. Mitigación: guardar el orden original y restaurarlo en cualquier respuesta distinta de 204.
- **Pestañas concurrentes.** Dos pestañas pueden enviar órdenes basados en listas distintas. Mitigación: el conjunto exacto de identificadores con `409`.
- **Alta concurrente.** Dos altas simultáneas pueden obtener la misma `MAX+1`. Con posiciones densas y una sola instancia es un riesgo bajo, y el desempate `created_at, id` mantiene un orden determinista. Si preocupa, usar `SELECT ... FOR UPDATE` sobre la tabla o una restricción única, pero sería un cambio de diseño que no se propone aquí.
- **Móvil.** Sin `touch-action: none` el scroll compite con el arrastre. Sin Pointer Events, HTML5 DnD no funciona en tacto. Mitigación: ambas cosas y verificación en emulación táctil.
- **Accesibilidad.** El asidero debe tener nombre accesible y ser alcanzable por teclado; el movimiento con `Alt+↑/↓` debe anunciarse. Mitigación: `aria-label` que incluye el título y, opcionalmente, un `aria-live` que diga "Movida a posición N de M".
- **Migración en producción.** `ADD COLUMN ... DEFAULT 0 NOT NULL` y el `UPDATE` bloquean la tabla brevemente; con pocas filas no es un problema. Antes de desplegar, hacer copia de seguridad, como para cualquier migración.
- **Recuperación.**
  - Código: revertir el commit de la funcionalidad. La columna puede quedar; no afecta a la versión anterior porque el `SELECT` antiguo no la usa y `Position` tiene valor por defecto.
  - Base de datos: ejecutar `down` (`DROP COLUMN position`); pierde el orden personalizado, que la versión anterior ya ignoraba.
  - Si la reordenación deja posiciones incoherentes, basta volver a normalizar con el `UPDATE ... row_number()` de la migración.
- **Incertidumbres pendientes:**
  1. ¿Se quiere el atajo de teclado `Alt+↑/↓` o basta con el ratón y el tacto? (Recomendación: incluirlo; coste bajo.)
  2. ¿Las tareas completadas deben seguir en la misma lista ordenable o separarse? (Recomendación: misma lista, sin cambios de comportamiento.)
  3. ¿Se acepta el índice `tasks_position_idx`? (Recomendación: sí, es barato; si no, omitirlo.)
  4. Confirmar en `actions/` cómo devuelve Buffalo un `204` sin cuerpo en esta versión, antes de implementar.
