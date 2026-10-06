import {
  Alert,
  Button,
  Group,
  Loader,
  Stack,
  Text,
  TextInput,
  Textarea,
  Title,
} from "@mantine/core";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ProblemError } from "../heart/api";
import { type SelfProfile, useSelfProfile, useUpdateSelfProfile } from "./api";

type Values = {
  preferred_name: string;
  birth_date: string;
  core_values: string;
  bio: string;
};

const empty: Values = { preferred_name: "", birth_date: "", core_values: "", bio: "" };

/** Self area: who you are, in your own words (SELF-01, SELF-03, SELF-04). */
export function SelfPage() {
  const { t } = useTranslation();
  const profile = useSelfProfile();

  if (profile.isPending) {
    return <Loader size="sm" aria-label={t("self.loading")} />;
  }
  if (profile.isError) {
    return (
      <Alert color="red" role="alert">
        {t("self.loadFailed")}
      </Alert>
    );
  }
  // Keyed by version: a save or a reload starts the form again from the server's copy.
  return <ProfileForm key={profile.data.version} profile={profile.data} />;
}

function ProfileForm({ profile }: { profile: SelfProfile }) {
  const { t } = useTranslation();
  const save = useUpdateSelfProfile();
  const [values, setValues] = useState<Values>(() => fromProfile(profile));
  const [message, setMessage] = useState<{ kind: "saved" | "error"; text: string } | null>(null);

  async function handleSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage(null);
    try {
      await save.mutateAsync({ version: profile.version, patch: toPatch(values) });
      setMessage({ kind: "saved", text: t("self.saved") });
    } catch (err) {
      const stale = err instanceof ProblemError && err.problem.status === 412;
      setMessage({ kind: "error", text: stale ? t("self.stale") : t("self.saveFailed") });
    }
  }

  function set(key: keyof Values, next: string) {
    setValues((v) => ({ ...v, [key]: next }));
  }

  return (
    <Stack gap="md" maw={640}>
      <div>
        <Title order={1}>{t("self.title")}</Title>
        <Text c="dimmed">{t("self.intro")}</Text>
      </div>

      <form onSubmit={(e) => void handleSubmit(e)}>
        <Stack gap="sm">
          {message && (
            <Alert
              color={message.kind === "saved" ? "teal" : "red"}
              role={message.kind === "error" ? "alert" : "status"}
            >
              {message.text}
            </Alert>
          )}
          <TextInput
            id="self-preferred-name"
            label={t("self.form.preferredName")}
            value={values.preferred_name}
            onChange={(e) => {
              set("preferred_name", e.currentTarget.value);
            }}
          />
          <TextInput
            id="self-birth-date"
            type="date"
            label={t("self.form.birthDate")}
            value={values.birth_date}
            onChange={(e) => {
              set("birth_date", e.currentTarget.value);
            }}
          />
          <Textarea
            id="self-core-values"
            label={t("self.form.coreValues")}
            description={t("self.form.coreValuesHint")}
            minRows={3}
            value={values.core_values}
            onChange={(e) => {
              set("core_values", e.currentTarget.value);
            }}
          />
          <Textarea
            id="self-bio"
            label={t("self.form.bio")}
            minRows={5}
            value={values.bio}
            onChange={(e) => {
              set("bio", e.currentTarget.value);
            }}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={save.isPending}>
              {t("common.save")}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  );
}

function fromProfile(p: SelfProfile): Values {
  return {
    preferred_name: p.preferred_name ?? empty.preferred_name,
    birth_date: p.birth_date ?? empty.birth_date,
    core_values: p.core_values ?? empty.core_values,
    bio: p.bio ?? empty.bio,
  };
}

/** The whole form as a merge patch: empty fields become null, which clears them. */
function toPatch(v: Values): Record<string, unknown> {
  const clean = (s: string) => (s.trim() === "" ? null : s.trim());
  return {
    preferred_name: clean(v.preferred_name),
    birth_date: clean(v.birth_date),
    core_values: clean(v.core_values),
    bio: clean(v.bio),
  };
}
