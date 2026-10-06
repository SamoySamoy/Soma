import { Button, Group, Stack, Textarea, TextInput } from "@mantine/core";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { type ContactInput, optionalFields } from "./api";

const birthdayPattern = /^\d{4}-\d{2}-\d{2}$/;

type Props = {
  initial: ContactInput;
  saving: boolean;
  /** Field messages from the server, keyed by field name. */
  serverErrors: Record<string, string>;
  onSubmit: (value: ContactInput) => void;
  onCancel: () => void;
};

/**
 * A contact form. Empty optional fields are sent as absent, not as "", and
 * the server treats those as unset.
 */
export function ContactForm({ initial, saving, serverErrors, onSubmit, onCancel }: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState<ContactInput>(initial);
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({});

  const errors = { ...serverErrors, ...localErrors };

  function set(key: keyof ContactInput, next: string) {
    setValue((v) => ({ ...v, [key]: next }));
  }

  function handleSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const next: Record<string, string> = {};
    if (!value.display_name.trim()) next.display_name = t("people.validation.nameRequired");
    const birthday = value.birthday?.trim() ?? "";
    if (birthday !== "" && !birthdayPattern.test(birthday)) {
      next.birthday = t("people.validation.birthday");
    }
    setLocalErrors(next);
    if (Object.keys(next).length > 0) return;

    const input: ContactInput = { display_name: value.display_name.trim() };
    for (const field of optionalFields) {
      const trimmed = value[field]?.trim();
      if (trimmed) input[field] = trimmed;
    }
    onSubmit(input);
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <Stack gap="sm">
        <TextInput
          id="contact-display-name"
          label={t("people.form.displayName")}
          value={value.display_name}
          onChange={(e) => {
            set("display_name", e.currentTarget.value);
          }}
          error={errors.display_name}
          required
          autoFocus
        />
        <TextInput
          id="contact-nickname"
          label={t("people.form.nickname")}
          value={value.nickname ?? ""}
          onChange={(e) => {
            set("nickname", e.currentTarget.value);
          }}
          error={errors.nickname}
        />
        <TextInput
          id="contact-email"
          type="email"
          label={t("people.form.email")}
          value={value.email ?? ""}
          onChange={(e) => {
            set("email", e.currentTarget.value);
          }}
          error={errors.email}
        />
        <TextInput
          id="contact-phone"
          type="tel"
          label={t("people.form.phone")}
          value={value.phone ?? ""}
          onChange={(e) => {
            set("phone", e.currentTarget.value);
          }}
          error={errors.phone}
        />
        <TextInput
          id="contact-birthday"
          label={t("people.form.birthday")}
          description={t("people.form.birthdayHint")}
          placeholder="1990-04-23"
          value={value.birthday ?? ""}
          onChange={(e) => {
            set("birthday", e.currentTarget.value);
          }}
          error={errors.birthday}
        />
        <TextInput
          id="contact-how-we-met"
          label={t("people.form.howWeMet")}
          value={value.how_we_met ?? ""}
          onChange={(e) => {
            set("how_we_met", e.currentTarget.value);
          }}
          error={errors.how_we_met}
        />
        <Textarea
          id="contact-notes"
          label={t("people.form.notes")}
          minRows={3}
          value={value.notes ?? ""}
          onChange={(e) => {
            set("notes", e.currentTarget.value);
          }}
          error={errors.notes}
        />
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel} disabled={saving}>
            {t("people.form.cancel")}
          </Button>
          <Button type="submit" loading={saving}>
            {saving ? t("people.form.saving") : t("people.form.save")}
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
