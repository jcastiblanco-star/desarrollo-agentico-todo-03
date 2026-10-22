// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  fetch(form.action, {
    method: "PUT",
    headers: {
      "X-CSRF-Token": form.querySelector('input[name="authenticity_token"]').value,
    },
    credentials: "same-origin",
  });
});

// Reordenar tareas con el asidero: arrastrar con ratón, lápiz o tacto,
// o mover la tarea enfocada con Alt+↑ / Alt+↓. Cada cambio envía la lista
// completa de identificadores a /tasks/reorder.
(function () {
  var list = document.querySelector(".task-list");
  if (!list) {
    return;
  }

  var status = document.getElementById("reorder-status");
  var errorBox = document.getElementById("reorder-error");
  var drag = null;

  function items() {
    return Array.prototype.slice.call(list.children);
  }

  function currentIds() {
    return items().map(function (li) {
      return li.dataset.taskId;
    });
  }

  function restore(ids) {
    ids.forEach(function (id) {
      var li = list.querySelector('[data-task-id="' + id + '"]');
      if (li) {
        list.appendChild(li);
      }
    });
  }

  function announce(li) {
    var position = items().indexOf(li) + 1;
    status.textContent = "Tarea movida a la posición " + position + " de " + items().length + ".";
  }

  function save(ids, previousIds, li) {
    var token = li.querySelector('input[name="authenticity_token"]').value;

    fetch("/tasks/reorder", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": token,
      },
      credentials: "same-origin",
      body: JSON.stringify({ ids: ids }),
    })
      .then(function (response) {
        if (response.status === 204) {
          errorBox.hidden = true;
          return;
        }
        if (response.status === 409) {
          // La lista cambió en otra pestaña: se muestra la versión del servidor.
          window.location.reload();
          return;
        }
        restore(previousIds);
        errorBox.hidden = false;
      })
      .catch(function () {
        restore(previousIds);
        errorBox.hidden = false;
      });
  }

  list.addEventListener("pointerdown", function (event) {
    var handle = event.target.closest(".task-handle");
    if (!handle || event.button !== 0) {
      return;
    }

    event.preventDefault();
    var li = handle.closest(".task");
    drag = {
      li: li,
      handle: handle,
      pointerId: event.pointerId,
      startY: event.clientY,
      startIds: currentIds(),
      // Desplazamiento de la tarea en el diseño cuando se intercambia con un hermano.
      layoutShift: 0,
    };
    handle.setPointerCapture(event.pointerId);
    li.classList.add("task-dragging");
  });

  list.addEventListener("pointermove", function (event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }

    var li = drag.li;
    li.style.transform = "translateY(" + (event.clientY - drag.startY - drag.layoutShift) + "px)";

    var moved = true;
    while (moved) {
      moved = false;
      var rect = li.getBoundingClientRect();
      var center = rect.top + rect.height / 2;
      var prev = li.previousElementSibling;
      var next = li.nextElementSibling;

      var step = null;
      if (prev && center < prev.getBoundingClientRect().top + prev.offsetHeight / 2) {
        step = function () { list.insertBefore(li, prev); };
      } else if (next && center > next.getBoundingClientRect().top + next.offsetHeight / 2) {
        step = function () { list.insertBefore(li, next.nextElementSibling); };
      }

      if (step) {
        var before = li.offsetTop;
        step();
        drag.layoutShift += li.offsetTop - before;
        li.style.transform = "translateY(" + (event.clientY - drag.startY - drag.layoutShift) + "px)";
        moved = true;
      }
    }
  });

  function endDrag(event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }

    var finished = drag;
    drag = null;
    finished.li.style.transform = "";
    finished.li.classList.remove("task-dragging");

    var ids = currentIds();
    if (ids.join() !== finished.startIds.join()) {
      announce(finished.li);
      save(ids, finished.startIds, finished.li);
    }
  }

  list.addEventListener("pointerup", endDrag);
  list.addEventListener("pointercancel", endDrag);

  list.addEventListener("keydown", function (event) {
    var handle = event.target.closest(".task-handle");
    if (!handle || !event.altKey) {
      return;
    }

    var li = handle.closest(".task");
    var startIds = currentIds();

    if (event.key === "ArrowUp" && li.previousElementSibling) {
      event.preventDefault();
      list.insertBefore(li, li.previousElementSibling);
    } else if (event.key === "ArrowDown" && li.nextElementSibling) {
      event.preventDefault();
      list.insertBefore(li, li.nextElementSibling.nextElementSibling);
    } else {
      return;
    }

    // Mover el nodo puede quitar el foco; se devuelve al asidero de la tarea.
    handle.focus();
    announce(li);
    save(currentIds(), startIds, li);
  });
})();
