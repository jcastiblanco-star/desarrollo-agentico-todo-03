package actions

import (
	"net/http"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TasksIndex muestra la lista de tareas y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	tasks := models.Tasks{}
	if err := tx.Order("position asc, created_at asc, id asc").All(&tasks); err != nil {
		return err
	}

	c.Set("tasks", tasks)
	c.Set("task", &models.Task{})
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/index.plush.html"))
}

// TasksCreate crea una tarea nueva a partir del formulario.
func TasksCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := c.Bind(task); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(task.Title)

	position, err := models.NextPosition(tx)
	if err != nil {
		return err
	}
	task.Position = position

	verrs, err := tx.ValidateAndCreate(task)
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		tasks := models.Tasks{}
		if err := tx.Order("position asc, created_at asc, id asc").All(&tasks); err != nil {
			return err
		}

		c.Set("tasks", tasks)
		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/index.plush.html"))
	}

	c.Flash().Add("success", "Tarea creada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksReorder guarda el orden completo de las tareas. El cuerpo JSON debe
// contener todos los identificadores, en el nuevo orden, sin repetirlos.
func TasksReorder(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	body := struct {
		IDs []uuid.UUID `json:"ids"`
	}{}
	if err := c.Bind(&body); err != nil || body.IDs == nil {
		return c.Render(http.StatusUnprocessableEntity, r.JSON(map[string]string{"error": "Se esperaba la lista de identificadores."}))
	}

	seen := make(map[uuid.UUID]bool, len(body.IDs))
	for _, id := range body.IDs {
		if seen[id] {
			return c.Render(http.StatusUnprocessableEntity, r.JSON(map[string]string{"error": "Hay identificadores repetidos."}))
		}
		seen[id] = true
	}

	tasks := models.Tasks{}
	if err := tx.All(&tasks); err != nil {
		return err
	}

	// Si la lista ha cambiado desde que el cliente la cargó (otra pestaña
	// ha creado o eliminado tareas), el orden recibido ya no es válido.
	if len(tasks) != len(body.IDs) {
		return c.Render(http.StatusConflict, r.JSON(map[string]string{"error": "La lista ha cambiado. Recarga la página."}))
	}
	existing := make(map[uuid.UUID]bool, len(tasks))
	for _, t := range tasks {
		existing[t.ID] = true
	}
	for _, id := range body.IDs {
		if !existing[id] {
			return c.Render(http.StatusConflict, r.JSON(map[string]string{"error": "La lista ha cambiado. Recarga la página."}))
		}
	}

	// SQL directo: no se quiere tocar updated_at ni revalidar el título de todas las filas.
	for i, id := range body.IDs {
		if err := tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", i, id).Exec(); err != nil {
			return err
		}
	}

	c.Response().WriteHeader(http.StatusNoContent)
	return nil
}

// TasksEdit muestra el formulario de edición de una tarea existente.
func TasksEdit(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	c.Set("task", task)
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/edit.plush.html"))
}

// TasksUpdate actualiza solo el título, conservando los demás datos de la tarea.
func TasksUpdate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	input := struct {
		Title string `form:"title" json:"title"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(input.Title)

	verrs, err := tx.ValidateAndUpdate(task)
	if err != nil {
		return err
	}
	if verrs.HasAny() {
		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/edit.plush.html"))
	}

	c.Flash().Add("success", "Tarea actualizada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksToggle alterna una tarea entre completada y pendiente.
func TasksToggle(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	task.Completed = !task.Completed
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksDestroy elimina una tarea.
func TasksDestroy(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea eliminada.")
	return c.Redirect(http.StatusSeeOther, "/")
}
