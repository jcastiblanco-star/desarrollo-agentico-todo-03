# Feature: Marcar en verde las tareas completadas y en rojo las pendientes

Issue: #58967

## Encargo y criterios de aceptación

### Requisitos explícitos (del encargo)

- Cada tarea terminada debe marcarse en **verde**.
- Cada tarea que no está hecha debe marcarse en **rojo**.
- **Aclaración del encargo:** el color afecta a **toda la fila** de la tarea, no solo al indicador del botón de completar.

### Criterios de aceptación

1. Una tarea completada tiene la fila con fondo verde suave y una franja verde en el borde izquierdo. Su indicador de completada (círculo) sigue relleno en verde con el check en blanco.
2. Una tarea pendiente tiene la fila con fondo rojo suave y una franja roja en el borde izquierdo. Su indicador (círculo vacío) tiene borde rojo.
3. El color se aplica al cargar la página (renderizado de servidor) y al completar o reabrir una tarea sin recargar (actualización optimista de `app.js`). No hace falta tocar JavaScript: ambos casos se controlan con la clase `task-completed` que ya alterna el script.
4. El texto de la tarea, el estado "Completada" (solo para lectores de pantalla), los `aria-*` y el orden de la lista no cambian.
5. El texto de la fila sigue legible sobre el fondo suave (contraste de texto WCAG AA, 4.5:1 o más) y el indicador cumple 3:1 o más.
6. Al arrastrar una tarea (`task-dragging`), mantiene su estado visual de arrastre (fondo blanco y sombra) y no pierde el color de su estado al soltarla.
7. El hover de las acciones (`.task-actions`) y el asidero no cambian de comportamiento.

### Inferencias (no pedidas explícitamente; pendientes de confirmar)

- **Franja izquierda:** además del fondo, una franja de 3 px en el borde izquierdo refuerza el estado y evita depender solo del fondo suave. Se implementa con `box-shadow: inset`, para no desplazar el contenido.
- **Tonos:** verde suave nuevo (`--color-success-soft`) y verde de marca nuevo (`--color-success`); para el rojo se reutilizan `--color-danger` y `--color-danger-soft` (ya existen).
- **Accesibilidad sin color:** el estado se comunica también por el check (completada) frente al círculo vacío (pendiente), por `aria-pressed` y por el `aria-label` del botón. El color es refuerzo.

## Comportamiento actual y propuesta

### Actual

- `templates/tasks/index.plush.html:17` añade la clase `task-completed` al `<li class="task">` cuando `t.Completed` es verdadero.
- `public/assets/app.css`:
  - `.task` (línea ~130): rejilla de tres columnas, `padding: 0.5rem 0` y borde inferior gris; sin fondo.
  - `.task-completed .task-title` (línea ~200): título gris tachado.
  - `.completion-icon` (línea ~150): borde `#858580`; `.task-completed .completion-icon` (~200): fondo `#777771`.
- `public/assets/app.js:15` alterna `task-completed` al pulsar, así que el color de la fila cambia sin recargar.
- `application.plush.html:7` carga `/assets/app.css?v=5`.
- `:root` no tiene tokens de verde.

### Propuesta

1. **Tokens nuevos en `:root`:**
   ```css
   --color-success: #2e7d32;
   --color-success-soft: #eaf5ec;
   ```
   Se reutilizan `--color-danger` (`#a52b25`) y `--color-danger-soft` (`#fcf0ec`).

2. **Fila pendiente (roja):** nueva regla sobre `.task`, que es el estado por defecto:
   ```css
   .task {
     background: var(--color-danger-soft);
     box-shadow: inset 3px 0 0 var(--color-danger);
   }
   ```
   Se añade a la regla `.task` existente (línea ~130), no como bloque nuevo. Las propiedades nuevas no chocan con `transition` (que vive en otra regla `.task`, línea ~420).

3. **Fila completada (verde):**
   ```css
   .task-completed {
     background: var(--color-success-soft);
     box-shadow: inset 3px 0 0 var(--color-success);
   }
   ```

4. **Indicador de pendiente:** cambiar el borde base de `.completion-icon` de `#858580` a `var(--color-danger)`.

5. **Indicador de completada:** `.task-completed .completion-icon` de `#777771` a `var(--color-success)` en fondo y borde.

6. **Hover del indicador de completada:** la regla `.task-toggle-button:hover .completion-icon` (0,3,0) pintaría la completada de rojo al pasar el ratón. Añadir:
   ```css
   .task-completed .task-toggle-button:hover .completion-icon {
     background: var(--color-success);
     border-color: var(--color-success);
     color: #ffffff;
   }
   ```

7. **Arrastre:** la regla `.task.task-dragging` (línea ~445) fija `background: #ffffff`. Debe ganar sobre el fondo de estado para que la fila arrastrada se vea como elevada. Como `.task-dragging` va después de `.task` y tiene la misma especificidad, gana por orden de cascada. La regla `.task-completed` (0,1,0) tiene la misma especificidad que `.task` pero va después: **si se declara `.task-completed` después de `.task.task-dragging` (0,2,0) no hay conflicto**, porque `.task.task-dragging` tiene más especificidad (0,2,0) que `.task-completed` (0,1,0). Verificar en el navegador que la fila arrastrada queda blanca también cuando está completada.

8. **Caché:** `templates/application.plush.html:7`: `?v=5` → `?v=6`.

No se modifican `app.js`, rutas Go, modelo, migraciones, tests ni la plantilla de la lista.

### Por qué esta forma

- **Clase de estado en la fila:** `task-completed` ya está en el `<li>` y el script la mantiene; el color de fila se deriva de ella sin lógica nueva.
- **Estado pendiente como valor por defecto:** `.task` pinta la fila en rojo y `.task-completed` la sobrescribe en verde, así que no hace falta una clase nueva para pendiente.
- **`box-shadow: inset`:** franja sin cambiar el ancho ni la posición del contenido.

## Alcance

### Incluido

- Tokens `--color-success` y `--color-success-soft` en `:root`.
- Fondo y franja de color en `.task` (pendiente, rojo) y `.task-completed` (completada, verde).
- Indicador de pendiente en rojo y de completada en verde.
- Regla de hover de la completada para que el indicador siga en verde.
- Comprobación de la fila arrastrada (`task-dragging`).
- Incremento de `?v=5` a `?v=6` en `templates/application.plush.html`.

### Excluido

- Cambiar el color del título o del texto de la tarea.
- Modo oscuro (la hoja actual no lo tiene).
- Cambios en `app.js`, `actions/tasks.go`, `models/task.go`, migraciones o tests.
- Tests automatizados de estilos: el repositorio no tiene infraestructura de tests de CSS ni de navegador.

## Diseño por capas

Solo capa de presentación:

- **Estilos (`public/assets/app.css`):** tokens, reglas de `.task`, `.task-completed`, indicadores y hover.
- **Plantilla de caché (`templates/application.plush.html`):** parámetro de versión.
- **Sin cambios en:** modelo, migraciones, acciones, rutas, `app.js`, plantilla de la lista.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.css` | Tokens en `:root` (~1-30); `.task` (~130); `.completion-icon` (~150); `.task-completed .completion-icon` (~200); `.task.task-dragging` (~445). |
| `templates/application.plush.html` | Línea 7: `app.css?v=5` → `?v=6`. |
| `templates/tasks/index.plush.html` | Solo lectura: clase `task-completed` en el `<li>` (línea 17). |
| `public/assets/app.js` | Solo lectura: alterna `task-completed` al completar o reabrir (línea 15). |
| `actions/tasks_test.go` | Solo lectura: los tests de la lista comprueban HTML, no estilos. |

## Implementación paso a paso

1. En `:root` de `public/assets/app.css`, añadir `--color-success: #2e7d32;` y `--color-success-soft: #eaf5ec;`.
2. En la regla `.task`, añadir `background: var(--color-danger-soft);` y `box-shadow: inset 3px 0 0 var(--color-danger);`.
3. Tras la regla `.task`, añadir `.task-completed { background: var(--color-success-soft); box-shadow: inset 3px 0 0 var(--color-success); }`.
4. En `.completion-icon`, cambiar `border: 1.5px solid #858580;` por `border: 1.5px solid var(--color-danger);`.
5. En `.task-completed .completion-icon`, cambiar `#777771` por `var(--color-success)` en `background` y `border-color`.
6. Tras la regla `.task-completed .completion-icon svg`, añadir la regla de hover del punto 6.
7. Verificar que `.task.task-dragging` (posterior) sigue fijando `background: #ffffff` y que gana sobre `.task-completed`.
8. En `templates/application.plush.html`, cambiar `?v=5` por `?v=6` en la línea del CSS.

## Tests y validación

### Comprobaciones del proyecto

En este orden:

1. `gofmt -l .` — debe devolver vacío (no hay cambios Go).
2. `go vet ./...`
3. `go build ./...`
4. `go test -p 1 ./...` — necesita PostgreSQL con migraciones de test. Si no está disponible, queda sin ejecutar y se informa; no se da por superada.
5. `git diff --check`

### Verificación visual (manual, obligatoria)

Con la aplicación en marcha (`/env:start`) y el navegador o `playwright-cli`:

1. Abrir `/`: las filas pendientes tienen fondo rojo suave y franja roja; las completadas, verde suave y franja verde. El indicador de completada es verde con check blanco; el de pendiente, borde rojo.
2. Pulsar **Completar** en una pendiente: la fila pasa a verde sin recargar.
3. Pulsar **Reabrir** en una completada: la fila vuelve a rojo sin recargar.
4. Recargar: los colores coinciden con el estado guardado.
5. Pasar el ratón sobre el indicador de una completada: sigue en verde. Sobre una pendiente: sigue en rojo.
6. Arrastrar una tarea completada y una pendiente: la fila arrastrada se ve blanca con sombra durante el arrastre.
7. Ancho ≤ 760 px: sin cambios de diseño ni desbordamiento.
8. Contraste: texto sobre fondo suave ≥ 4.5:1; franja e indicadores ≥ 3:1.

### Criterio de aceptación de la verificación

Los puntos 1 a 8 se cumplen.

## Riesgos y recuperación

- **Legibilidad del texto:** el fondo rojo suave es claro; el título gris tachado de las completadas puede quedar bajo contraste. Comprobar el punto 8 y, si falla, oscurecer `--color-muted` solo para `.task-completed .task-title`, como cambio aparte.
- **Arrastre:** si `.task-completed` gana al fondo blanco de `task-dragging`, la fila arrastrada quedaría verde. Mitigado con el punto 7 y la verificación 6.
- **Caché:** sin `?v=6` algunos navegadores mostrarían el estilo anterior. Mitigado con el paso 8.
- **Daltonismo:** rojo y verde pueden confundirse. Mitigación: el check frente al círculo vacío y `aria-pressed`.
- **Hover de completada:** sin la regla del punto 6, el indicador pasaría a rojo al pasar el ratón.
- **Colisión con cambios sin commitear:** `git status` muestra solo archivos sin seguimiento; no hay cambios pendientes en `app.css` ni en `application.plush.html`. Revisar si aparecen antes de implementar.
- **Recuperación:** solo CSS y una línea de plantilla; revertir el commit restaura el estado anterior sin efectos en datos.
