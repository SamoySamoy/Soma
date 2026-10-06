import { Alert, Badge, Button, Group, Loader, Paper, Stack, Text, TextInput } from "@mantine/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { ProblemError } from "../heart/api";
import { type BudgetLine, useBudgets, useSetBudget } from "./api";
import { formatMoney, fromMinor, monthOf, toMinor } from "./money";

type DigitsOf = (code: string) => number;

/** A month's budgets: set what each category may spend, see what has been spent (FIN-04). */
export function BudgetsTab({ base, digitsOf }: { base: string; digitsOf: DigitsOf }) {
  const { t } = useTranslation();
  const [month, setMonth] = useState(() => monthOf(new Date()));
  const budgets = useBudgets(month);
  const digits = digitsOf(base);

  return (
    <Stack gap="md">
      <TextInput
        id="budget-month"
        type="date"
        label={t("money.budgets.month")}
        description={t("money.budgets.monthHint")}
        value={month}
        onChange={(e) => {
          const day = e.currentTarget.value;
          if (day) setMonth(`${day.slice(0, 7)}-01`);
        }}
        maw={260}
      />
      {budgets.isPending && <Loader size="sm" aria-label={t("money.loading")} />}
      {budgets.isError && (
        <Alert color="red" role="alert">
          {budgets.error instanceof ProblemError &&
          budgets.error.problem.code === "money.missing_rate"
            ? budgets.error.message
            : t("money.loadFailed")}
        </Alert>
      )}
      {(budgets.data?.items ?? []).map((line) => (
        <BudgetRow
          key={`${line.category_id}-${month}`}
          line={line}
          month={month}
          base={base}
          digits={digits}
        />
      ))}
    </Stack>
  );
}

function BudgetRow({
  line,
  month,
  base,
  digits,
}: {
  line: BudgetLine;
  month: string;
  base: string;
  digits: number;
}) {
  const { t } = useTranslation();
  const save = useSetBudget(month);
  const [value, setValue] = useState(line.budget_minor ? fromMinor(line.budget_minor, digits) : "");
  const [message, setMessage] = useState<string | null>(null);
  const over = line.budget_minor > 0 && line.spent_minor > line.budget_minor;
  const indent = line.parent_id ? 24 : 0;

  async function handleSave() {
    setMessage(null);
    const amount = value.trim() === "" ? 0 : toMinor(value, digits);
    if (amount === null) {
      setMessage(t("money.amountInvalid"));
      return;
    }
    try {
      await save.mutateAsync({ categoryId: line.category_id, amountMinor: amount });
      setMessage(t("money.budgets.saved"));
    } catch {
      setMessage(t("money.saveFailed"));
    }
  }

  return (
    <Paper withBorder radius="md" p="sm" style={{ marginLeft: indent }}>
      <Group justify="space-between" align="flex-end" wrap="wrap" gap="sm">
        <Stack gap={2} style={{ minWidth: 160 }}>
          <Group gap="xs">
            <Text fw={line.parent_id ? 400 : 600}>{line.name}</Text>
            {over && (
              <Badge color="red" variant="light">
                {t("money.budgets.over")}
              </Badge>
            )}
          </Group>
          <Text size="sm" c="dimmed" style={{ fontVariantNumeric: "tabular-nums" }}>
            {t("money.budgets.spent", { amount: formatMoney(line.spent_minor, base, digits) })}
            {line.budget_minor > 0 &&
              ` · ${t("money.budgets.left", { amount: formatMoney(line.budget_minor - line.spent_minor, base, digits) })}`}
          </Text>
        </Stack>
        <Group gap="xs" align="flex-end">
          <TextInput
            aria-label={t("money.budgets.amountFor", { name: line.name, currency: base })}
            placeholder={t("money.budgets.none")}
            value={value}
            onChange={(e) => {
              setValue(e.currentTarget.value);
            }}
            inputMode="decimal"
            w={150}
          />
          <Button variant="default" onClick={() => void handleSave()} loading={save.isPending}>
            {t("common.save")}
          </Button>
        </Group>
      </Group>
      {message && (
        <Text size="sm" role="status" mt={4}>
          {message}
        </Text>
      )}
    </Paper>
  );
}
