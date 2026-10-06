import {
  Alert,
  Badge,
  Button,
  Drawer,
  Group,
  Loader,
  Paper,
  Select,
  Stack,
  Text,
  TextInput,
  Textarea,
  Title,
} from "@mantine/core";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ConfirmAction } from "../../components/ConfirmAction";
import { ProblemError } from "../heart/api";
import {
  type JournalEntry,
  type JournalEntryInput,
  useCreateJournalEntry,
  useDeleteJournalEntry,
  useJournalEntries,
  useUpdateJournalEntry,
} from "./api";

const today = () => new Date().toISOString().slice(0, 10);
const excerptLength = 240;

type Editing = { mode: "create" } | { mode: "edit"; entry: JournalEntry };

/** Mind area: the owner's private journal (JRN-01..04). */
export function JournalPage() {
  const { t } = useTranslation();
  const entries = useJournalEntries();
  const create = useCreateJournalEntry();
  const update = useUpdateJournalEntry();
  const remove = useDeleteJournalEntry();

  const [editing, setEditing] = useState<Editing | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const items = entries.data?.pages.flatMap((page) => page.items) ?? [];

  async function save(input: JournalEntryInput) {
    setFormError(null);
    try {
      if (editing?.mode === "edit") {
        await update.mutateAsync({
          id: editing.entry.id,
          version: editing.entry.version,
          patch: toPatch(input),
        });
      } else {
        await create.mutateAsync(input);
      }
      setEditing(null);
    } catch (err) {
      setFormError(
        err instanceof ProblemError && err.problem.status === 412
          ? t("journal.stale")
          : t("journal.saveFailed"),
      );
    }
  }

  async function trash(entry: JournalEntry) {
    try {
      await remove.mutateAsync(entry.id);
      setEditing(null);
    } catch {
      setFormError(t("journal.saveFailed"));
    }
  }

  return (
    <Stack gap="md">
      <Group justify="space-between" align="flex-end">
        <div>
          <Title order={1}>{t("journal.title")}</Title>
          <Text c="dimmed">{t("journal.intro")}</Text>
        </div>
        <Button
          onClick={() => {
            setEditing({ mode: "create" });
          }}
        >
          {t("journal.write")}
        </Button>
      </Group>

      {entries.isPending && <Loader size="sm" aria-label={t("journal.loading")} />}
      {entries.isError && (
        <Alert color="red" role="alert">
          {t("journal.loadFailed")}
        </Alert>
      )}
      {entries.isSuccess && items.length === 0 && <Text c="dimmed">{t("journal.empty")}</Text>}

      <Stack gap="sm">
        {items.map((entry) => (
          <Paper key={entry.id} withBorder radius="md" p="md">
            <Group justify="space-between" align="flex-start" wrap="nowrap">
              <Stack gap={4} style={{ minWidth: 0 }}>
                <Group gap="xs">
                  <Text size="sm" c="dimmed" style={{ fontVariantNumeric: "tabular-nums" }}>
                    {entry.entry_date}
                  </Text>
                  {entry.mood !== undefined && (
                    <Badge variant="light">{t("journal.moodLabel", { mood: entry.mood })}</Badge>
                  )}
                </Group>
                {entry.title && <Text fw={600}>{entry.title}</Text>}
                <Text style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
                  {excerpt(entry.body)}
                </Text>
              </Stack>
              <Button
                variant="subtle"
                size="xs"
                aria-label={t("journal.editNamed", { date: entry.entry_date })}
                onClick={() => {
                  setEditing({ mode: "edit", entry });
                }}
              >
                {t("journal.edit")}
              </Button>
            </Group>
          </Paper>
        ))}
      </Stack>

      {entries.hasNextPage && (
        <Group>
          <Button
            variant="default"
            onClick={() => void entries.fetchNextPage()}
            loading={entries.isFetchingNextPage}
          >
            {t("journal.loadMore")}
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
        size="lg"
        title={editing?.mode === "edit" ? t("journal.form.editTitle") : t("journal.form.addTitle")}
      >
        {editing && (
          <EntryForm
            key={editing.mode === "edit" ? editing.entry.id : "create"}
            initial={
              editing.mode === "edit" ? toInput(editing.entry) : { entry_date: today(), body: "" }
            }
            saving={create.isPending || update.isPending}
            error={formError}
            onSubmit={(input) => void save(input)}
            onCancel={() => {
              setEditing(null);
            }}
            trash={editing.mode === "edit" ? () => void trash(editing.entry) : undefined}
            trashBusy={remove.isPending}
          />
        )}
      </Drawer>
    </Stack>
  );
}

function EntryForm({
  initial,
  saving,
  error,
  onSubmit,
  onCancel,
  trash,
  trashBusy,
}: {
  initial: JournalEntryInput;
  saving: boolean;
  error: string | null;
  onSubmit: (input: JournalEntryInput) => void;
  onCancel: () => void;
  trash?: (() => void) | undefined;
  trashBusy: boolean;
}) {
  const { t } = useTranslation();
  const [value, setValue] = useState<JournalEntryInput>(initial);
  const [localError, setLocalError] = useState<string | null>(null);

  function handleSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (value.body.trim() === "" && (value.title ?? "").trim() === "") {
      setLocalError(t("journal.validation.empty"));
      return;
    }
    setLocalError(null);
    const input: JournalEntryInput = { entry_date: value.entry_date, body: value.body.trim() };
    const title = value.title?.trim();
    if (title) input.title = title;
    if (value.mood !== undefined) input.mood = value.mood;
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
          id="entry-date"
          type="date"
          label={t("journal.form.date")}
          value={value.entry_date}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => ({ ...v, entry_date: next }));
          }}
          required
        />
        <TextInput
          id="entry-title"
          label={t("journal.form.titleLabel")}
          value={value.title ?? ""}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => ({ ...v, title: next }));
          }}
        />
        <Select
          id="entry-mood"
          label={t("journal.form.mood")}
          data={[
            { value: "", label: t("journal.form.noMood") },
            ...[1, 2, 3, 4, 5].map((n) => ({
              value: String(n),
              label: t("journal.moodLabel", { mood: n }),
            })),
          ]}
          value={value.mood === undefined ? "" : String(value.mood)}
          onChange={(next) => {
            setValue((v) => withMood(v, next ? Number(next) : undefined));
          }}
          allowDeselect={false}
        />
        <Textarea
          id="entry-body"
          label={t("journal.form.body")}
          minRows={10}
          value={value.body}
          onChange={(e) => {
            const next = e.currentTarget.value;
            setValue((v) => ({ ...v, body: next }));
          }}
        />
        <Group justify="space-between">
          {trash ? (
            <ConfirmAction
              label={t("journal.trash.action")}
              question={t("journal.trash.confirm")}
              confirmLabel={t("journal.trash.confirmAction")}
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

/** Sets the mood, or removes it when none is chosen. */
function withMood(v: JournalEntryInput, mood: number | undefined): JournalEntryInput {
  const out = { ...v };
  if (mood === undefined) {
    delete out.mood;
  } else {
    out.mood = mood;
  }
  return out;
}

function excerpt(body: string): string {
  return body.length > excerptLength ? `${body.slice(0, excerptLength).trimEnd()}…` : body;
}

function toInput(e: JournalEntry): JournalEntryInput {
  const input: JournalEntryInput = { entry_date: e.entry_date, body: e.body };
  if (e.title) input.title = e.title;
  if (e.mood !== undefined) input.mood = e.mood;
  return input;
}

/** Full form as a merge patch: cleared fields become null, so the server clears them. */
function toPatch(input: JournalEntryInput): Record<string, unknown> {
  return {
    entry_date: input.entry_date,
    body: input.body,
    title: input.title ?? null,
    mood: input.mood ?? null,
  };
}
