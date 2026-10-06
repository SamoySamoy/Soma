import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Drawer,
  Group,
  Loader,
  Paper,
  Stack,
  Text,
  TextInput,
  Textarea,
  Title,
} from "@mantine/core";
import { useMemo, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ConfirmAction } from "../../components/ConfirmAction";
import { ProblemError } from "../heart/api";
import {
  type Task,
  type TaskInput,
  useCreateTask,
  useDeleteTask,
  useSetTaskDone,
  useTasks,
  useUpdateTask,
} from "./api";

type Editing = { mode: "create" } | { mode: "edit"; task: Task };

/** Responsibilities area: things to do (TSK-01..02). */
export function TasksPage() {
  const { t } = useTranslation();
  const tasks = useTasks();
  const create = useCreateTask();
  const update = useUpdateTask();
  const setDone = useSetTaskDone();
  const remove = useDeleteTask();

  const [editing, setEditing] = useState<Editing | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [showDone, setShowDone] = useState(false);

  const all = useMemo(() => tasks.data?.pages.flatMap((page) => page.items) ?? [], [tasks.data]);
  const visible = useMemo(
    () => sortTasks(all.filter((task) => showDone || !task.completed)),
    [all, showDone],
  );

  async function save(input: TaskInput) {
    setFormError(null);
    try {
      if (editing?.mode === "edit") {
        await update.mutateAsync({
          id: editing.task.id,
          version: editing.task.version,
          patch: toPatch(input),
        });
      } else {
        await create.mutateAsync(input);
      }
      setEditing(null);
    } catch (err) {
      setFormError(
        err instanceof ProblemError && err.problem.status === 412
          ? t("tasks.stale")
          : t("tasks.saveFailed"),
      );
    }
  }

  async function trash(task: Task) {
    try {
      await remove.mutateAsync(task.id);
      setEditing(null);
    } catch {
      setFormError(t("tasks.saveFailed"));
    }
  }

  return (
    <Stack gap="md">
      <Group justify="space-between" align="flex-end">
        <div>
          <Title order={1}>{t("tasks.title")}</Title>
          <Text c="dimmed">{t("tasks.intro")}</Text>
        </div>
        <Button
          onClick={() => {
            setEditing({ mode: "create" });
          }}
        >
          {t("tasks.add")}
        </Button>
      </Group>

      <Checkbox
        label={t("tasks.showDone")}
        checked={showDone}
        onChange={(e) => {
          setShowDone(e.currentTarget.checked);
        }}
      />

      {tasks.isPending && <Loader size="sm" aria-label={t("tasks.loading")} />}
      {tasks.isError && (
        <Alert color="red" role="alert">
          {t("tasks.loadFailed")}
        </Alert>
      )}
      {tasks.isSuccess && visible.length === 0 && <Text c="dimmed">{t("tasks.empty")}</Text>}

      <Stack gap="sm">
        {visible.map((task) => (
          <Paper key={task.id} withBorder radius="md" p="sm">
            <Group justify="space-between" wrap="nowrap" align="center">
              <Group gap="sm" wrap="nowrap" style={{ minWidth: 0 }}>
                <Checkbox
                  checked={task.completed}
                  aria-label={t(task.completed ? "tasks.reopenNamed" : "tasks.completeNamed", {
                    title: task.title,
                  })}
                  onChange={() => {
                    setDone.mutate({ id: task.id, done: !task.completed });
                  }}
                />
                <Stack gap={2} style={{ minWidth: 0 }}>
                  <Text
                    fw={500}
                    td={task.completed ? "line-through" : "none"}
                    c={task.completed ? "dimmed" : "bright"}
                    style={{ overflowWrap: "anywhere" }}
                  >
                    {task.title}
                  </Text>
                  <Group gap="xs">
                    {task.due_on && (
                      <Text size="sm" c="dimmed" style={{ fontVariantNumeric: "tabular-nums" }}>
                        {t("tasks.due", { date: task.due_on })}
                      </Text>
                    )}
                    {task.overdue && (
                      <Badge color="red" variant="light">
                        {t("tasks.overdue")}
                      </Badge>
                    )}
                  </Group>
                </Stack>
              </Group>
              <Button
                variant="subtle"
                size="xs"
                aria-label={t("tasks.editNamed", { title: task.title })}
                onClick={() => {
                  setEditing({ mode: "edit", task });
                }}
              >
                {t("tasks.edit")}
              </Button>
            </Group>
          </Paper>
        ))}
      </Stack>

      {tasks.hasNextPage && (
        <Group>
          <Button
            variant="default"
            onClick={() => void tasks.fetchNextPage()}
            loading={tasks.isFetchingNextPage}
          >
            {t("tasks.loadMore")}
          </Button>
        </Group>
      )}

      <Drawer
        opened={editing !== null}
        onClose={() => {
          setEditing(null);
          setFormError(null);
        }}
        position="right"
        size="md"
        title={editing?.mode === "edit" ? t("tasks.form.editTitle") : t("tasks.form.addTitle")}
      >
        {editing && (
          <TaskForm
            key={editing.mode === "edit" ? editing.task.id : "create"}
            initial={editing.mode === "edit" ? toInput(editing.task) : { title: "" }}
            saving={create.isPending || update.isPending}
            error={formError}
            onSubmit={(input) => void save(input)}
            onCancel={() => {
              setEditing(null);
            }}
            trash={editing.mode === "edit" ? () => void trash(editing.task) : undefined}
            trashBusy={remove.isPending}
          />
        )}
      </Drawer>
    </Stack>
  );
}

function TaskForm({
  initial,
  saving,
  error,
  onSubmit,
  onCancel,
  trash,
  trashBusy,
}: {
  initial: TaskInput;
  saving: boolean;
  error: string | null;
  onSubmit: (input: TaskInput) => void;
  onCancel: () => void;
  trash?: (() => void) | undefined;
  trashBusy: boolean;
}) {
  const { t } = useTranslation();
  const [value, setValue] = useState<TaskInput>(initial);
  const [localError, setLocalError] = useState<string | null>(null);

  function handleSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (value.title.trim() === "") {
      setLocalError(t("tasks.validation.title"));
      return;
    }
    setLocalError(null);
    const input: TaskInput = { title: value.title.trim() };
    const notes = value.notes?.trim();
    if (notes) input.notes = notes;
    if (value.due_on) input.due_on = value.due_on;
    onSubmit(input);
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <Stack gap="sm">
        {(error ?? localError) && (
          <Alert color="red" role="alert">
            {error ?? localError}
          </Alert>
        )}
        <TextInput
          id="task-title"
          label={t("tasks.form.titleLabel")}
          value={value.title}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => ({ ...v, title: next }));
          }}
          required
          autoFocus
        />
        <TextInput
          id="task-due"
          type="date"
          label={t("tasks.form.due")}
          value={value.due_on ?? ""}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => withDueOn(v, next));
          }}
        />
        <Textarea
          id="task-notes"
          label={t("tasks.form.notes")}
          minRows={3}
          value={value.notes ?? ""}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => ({ ...v, notes: next }));
          }}
        />
        <Group justify="space-between">
          {trash ? (
            <ConfirmAction
              label={t("tasks.trash.action")}
              question={t("tasks.trash.confirm")}
              confirmLabel={t("tasks.trash.confirmAction")}
              busy={trashBusy}
              onConfirm={trash}
            />
          ) : (
            <span />
          )}
          <Group gap="xs">
            <Button variant="default" onClick={onCancel} disabled={saving}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={saving}>
              {t("common.save")}
            </Button>
          </Group>
        </Group>
      </Stack>
    </form>
  );
}

/** Sets the due date, or removes it when cleared. */
function withDueOn(v: TaskInput, next: string): TaskInput {
  const out: TaskInput = { ...v };
  if (next === "") {
    delete out.due_on;
  } else {
    out.due_on = next;
  }
  return out;
}

/** Open tasks first, by due date (no date last), then done tasks. */
function sortTasks(tasks: Task[]): Task[] {
  return [...tasks].sort((a, b) => {
    if (a.completed !== b.completed) return a.completed ? 1 : -1;
    const ad = a.due_on ?? "9999-12-31";
    const bd = b.due_on ?? "9999-12-31";
    return ad < bd ? -1 : ad > bd ? 1 : a.title.localeCompare(b.title);
  });
}

function toInput(task: Task): TaskInput {
  const input: TaskInput = { title: task.title };
  if (task.notes) input.notes = task.notes;
  if (task.due_on) input.due_on = task.due_on;
  return input;
}

/** The whole form as a merge patch: cleared fields become null. */
function toPatch(input: TaskInput): Record<string, unknown> {
  return { title: input.title, notes: input.notes ?? null, due_on: input.due_on ?? null };
}
