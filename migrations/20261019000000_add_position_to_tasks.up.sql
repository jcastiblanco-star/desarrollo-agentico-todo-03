ALTER TABLE tasks ADD COLUMN position integer NOT NULL DEFAULT 0;

UPDATE tasks AS t
SET position = o.pos
FROM (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC, id ASC) - 1 AS pos
  FROM tasks
) AS o
WHERE t.id = o.id;

CREATE INDEX tasks_position_idx ON tasks (position);
