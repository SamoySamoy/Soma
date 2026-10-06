import { Button, Group, Stack, Text } from "@mantine/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";

type Props = {
  /** Button label that starts the action, for example "Move to trash". */
  label: string;
  /** Question shown before the action runs. */
  question: string;
  /** Label of the button that confirms. */
  confirmLabel: string;
  busy: boolean;
  onConfirm: () => void;
};

/**
 * A destructive action that needs a second click. Used for moving things to
 * the trash, so a stray click never removes anything.
 */
export function ConfirmAction({ label, question, confirmLabel, busy, onConfirm }: Props) {
  const { t } = useTranslation();
  const [confirming, setConfirming] = useState(false);

  if (!confirming) {
    return (
      <Button
        variant="subtle"
        color="red"
        onClick={() => {
          setConfirming(true);
        }}
      >
        {label}
      </Button>
    );
  }
  return (
    <Stack gap="xs">
      <Text size="sm">{question}</Text>
      <Group gap="xs">
        <Button color="red" onClick={onConfirm} loading={busy}>
          {confirmLabel}
        </Button>
        <Button
          variant="default"
          onClick={() => {
            setConfirming(false);
          }}
          disabled={busy}
        >
          {t("common.keep")}
        </Button>
      </Group>
    </Stack>
  );
}
