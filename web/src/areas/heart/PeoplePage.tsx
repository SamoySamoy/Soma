import { Alert, Button, Drawer, Group, Loader, Stack, Table, Text, Title } from "@mantine/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import {
  type Contact,
  type ContactInput,
  type ContactPatch,
  ProblemError,
  optionalFields,
  useContacts,
  useCreateContact,
  useTrashContact,
  useUpdateContact,
} from "./api";
import { ContactForm } from "./ContactForm";

type Editing = { mode: "create" } | { mode: "edit"; contact: Contact };

/** Heart area: the people the owner knows (PPL-01). */
export function PeoplePage() {
  const { t } = useTranslation();
  const contacts = useContacts();
  const create = useCreateContact();
  const update = useUpdateContact();
  const trash = useTrashContact();

  const [editing, setEditing] = useState<Editing | null>(null);
  const [serverErrors, setServerErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [confirmingTrash, setConfirmingTrash] = useState(false);

  const items = contacts.data?.pages.flatMap((page) => page.items) ?? [];

  function close() {
    setEditing(null);
    setServerErrors({});
    setFormError(null);
    setConfirmingTrash(false);
  }

  async function save(input: ContactInput) {
    setServerErrors({});
    setFormError(null);
    try {
      if (editing?.mode === "edit") {
        await update.mutateAsync({
          id: editing.contact.id,
          version: editing.contact.version,
          patch: toPatch(input),
        });
      } else {
        await create.mutateAsync(input);
      }
      close();
    } catch (err) {
      if (err instanceof ProblemError) {
        if (err.problem.status === 412) {
          setFormError(t("people.stale"));
          return;
        }
        setServerErrors(err.problem.errors ?? {});
      }
      setFormError(t("people.saveFailed"));
    }
  }

  async function moveToTrash(contact: Contact) {
    try {
      await trash.mutateAsync(contact.id);
      close();
    } catch {
      setFormError(t("people.saveFailed"));
    }
  }

  return (
    <Stack gap="md">
      <Group justify="space-between" align="flex-end">
        <div>
          <Title order={1}>{t("people.title")}</Title>
          <Text c="dimmed">{t("people.intro")}</Text>
        </div>
        <Button
          onClick={() => {
            setEditing({ mode: "create" });
          }}
        >
          {t("people.add")}
        </Button>
      </Group>

      {contacts.isPending && <Loader size="sm" aria-label={t("people.loading")} />}

      {contacts.isError && (
        <Alert color="red" role="alert">
          {t("people.loadFailed")}
        </Alert>
      )}

      {contacts.isSuccess && items.length === 0 && <Text c="dimmed">{t("people.empty")}</Text>}

      {items.length > 0 && (
        <div style={{ overflowX: "auto" }}>
          <Table striped highlightOnHover verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t("people.columns.name")}</Table.Th>
                <Table.Th>{t("people.columns.phone")}</Table.Th>
                <Table.Th>{t("people.columns.email")}</Table.Th>
                <Table.Th>{t("people.columns.birthday")}</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {items.map((contact) => (
                <Table.Tr key={contact.id}>
                  <Table.Td>
                    {contact.display_name}
                    {contact.nickname && (
                      <Text span c="dimmed">
                        {" "}
                        ({contact.nickname})
                      </Text>
                    )}
                  </Table.Td>
                  <Table.Td style={{ fontVariantNumeric: "tabular-nums" }}>
                    {contact.phone ?? ""}
                  </Table.Td>
                  <Table.Td>{contact.email ?? ""}</Table.Td>
                  <Table.Td style={{ fontVariantNumeric: "tabular-nums" }}>
                    {contact.birthday ?? ""}
                  </Table.Td>
                  <Table.Td>
                    <Button
                      variant="subtle"
                      size="xs"
                      aria-label={t("people.editNamed", { name: contact.display_name })}
                      onClick={() => {
                        setEditing({ mode: "edit", contact });
                      }}
                    >
                      {t("people.edit")}
                    </Button>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </div>
      )}

      {contacts.hasNextPage && (
        <Group>
          <Button
            variant="default"
            onClick={() => void contacts.fetchNextPage()}
            loading={contacts.isFetchingNextPage}
          >
            {t("people.loadMore")}
          </Button>
        </Group>
      )}

      <Drawer
        opened={editing !== null}
        onClose={close}
        position="right"
        size="md"
        title={editing?.mode === "edit" ? t("people.form.editTitle") : t("people.form.addTitle")}
      >
        {editing && (
          <Stack gap="lg">
            {formError && (
              <Alert color="red" role="alert">
                {formError}
              </Alert>
            )}
            <ContactForm
              key={editing.mode === "edit" ? editing.contact.id : "create"}
              initial={editing.mode === "edit" ? toInput(editing.contact) : { display_name: "" }}
              saving={create.isPending || update.isPending}
              serverErrors={serverErrors}
              onSubmit={(input) => void save(input)}
              onCancel={close}
            />
            {editing.mode === "edit" && (
              <TrashControl
                name={editing.contact.display_name}
                confirming={confirmingTrash}
                busy={trash.isPending}
                onStart={() => {
                  setConfirmingTrash(true);
                }}
                onConfirm={() => void moveToTrash(editing.contact)}
              />
            )}
          </Stack>
        )}
      </Drawer>
    </Stack>
  );
}

function TrashControl({
  name,
  confirming,
  busy,
  onStart,
  onConfirm,
}: {
  name: string;
  confirming: boolean;
  busy: boolean;
  onStart: () => void;
  onConfirm: () => void;
}) {
  const { t } = useTranslation();
  if (!confirming) {
    return (
      <Group>
        <Button variant="subtle" color="red" onClick={onStart}>
          {t("people.trash.action")}
        </Button>
      </Group>
    );
  }
  return (
    <Stack gap="xs">
      <Text size="sm">{t("people.trash.confirm", { name })}</Text>
      <Group>
        <Button color="red" onClick={onConfirm} loading={busy}>
          {t("people.trash.confirmAction")}
        </Button>
      </Group>
    </Stack>
  );
}

/** The form's view of a stored contact. */
function toInput(c: Contact): ContactInput {
  const input: ContactInput = { display_name: c.display_name };
  for (const field of optionalFields) {
    const v = c[field];
    if (v) input[field] = v;
  }
  return input;
}

/**
 * Builds a merge patch from the whole form. Fields the form left empty become
 * null, which clears them on the server; a merge patch would otherwise keep them.
 */
function toPatch(input: ContactInput): ContactPatch {
  const patch: ContactPatch = { display_name: input.display_name };
  for (const field of optionalFields) {
    patch[field] = input[field] ?? null;
  }
  return patch;
}
