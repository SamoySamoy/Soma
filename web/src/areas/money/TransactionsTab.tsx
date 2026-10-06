import {
  Alert,
  Badge,
  Button,
  Drawer,
  Group,
  Loader,
  Paper,
  Select,
  SegmentedControl,
  Stack,
  Text,
  TextInput,
  Textarea,
} from "@mantine/core";
import { useMemo, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ConfirmAction } from "../../components/ConfirmAction";
import { ProblemError } from "../heart/api";
import {
  type TransactionInput,
  useAccounts,
  useCategories,
  useDeleteTransaction,
  useRecordTransaction,
  useTransactions,
} from "./api";
import { formatMoney, toMinor, todayText } from "./money";

type Kind = "income" | "expense" | "transfer";
type DigitsOf = (code: string) => number;

/** Income, expenses and transfers, newest first (FIN-02). */
export function TransactionsTab({ base, digitsOf }: { base: string; digitsOf: DigitsOf }) {
  const { t } = useTranslation();
  const transactions = useTransactions();
  const accounts = useAccounts();
  const categories = useCategories();
  const remove = useDeleteTransaction();
  const [recording, setRecording] = useState(false);

  const accountName = useMemo(
    () => new Map((accounts.data ?? []).map((a) => [a.id, a])),
    [accounts.data],
  );
  const categoryName = useMemo(
    () => new Map((categories.data ?? []).map((c) => [c.id, c.name])),
    [categories.data],
  );
  const items = transactions.data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <Stack gap="md">
      <Group justify="flex-end">
        <Button
          onClick={() => {
            setRecording(true);
          }}
          disabled={(accounts.data ?? []).length === 0}
        >
          {t("money.tx.record")}
        </Button>
      </Group>
      {(accounts.data ?? []).length === 0 && accounts.isSuccess && (
        <Text c="dimmed">{t("money.tx.needAccount")}</Text>
      )}
      {transactions.isPending && <Loader size="sm" aria-label={t("money.loading")} />}
      {transactions.isError && (
        <Alert color="red" role="alert">
          {t("money.loadFailed")}
        </Alert>
      )}
      {transactions.isSuccess && items.length === 0 && (
        <Text c="dimmed">{t("money.tx.empty")}</Text>
      )}
      {items.map((tx) => {
        const account = accountName.get(tx.account_id);
        const currency = account?.currency ?? base;
        const label = tx.transfer_id
          ? t("money.tx.transfer")
          : tx.category_id
            ? categoryName.get(tx.category_id)
            : t(`money.tx.kind.${tx.kind}`);
        return (
          <Paper key={tx.id} withBorder radius="md" p="sm">
            <Group justify="space-between" wrap="nowrap" align="flex-start">
              <Stack gap={2} style={{ minWidth: 0 }}>
                <Group gap="xs">
                  <Text size="sm" c="dimmed" style={{ fontVariantNumeric: "tabular-nums" }}>
                    {tx.occurred_on}
                  </Text>
                  <Badge variant="light">{label}</Badge>
                </Group>
                <Text fw={500} style={{ overflowWrap: "anywhere" }}>
                  {tx.payee ?? account?.name ?? ""}
                </Text>
                {tx.notes && (
                  <Text size="sm" c="dimmed">
                    {tx.notes}
                  </Text>
                )}
              </Stack>
              <Stack gap={4} align="flex-end">
                <Text
                  fw={600}
                  c={tx.amount_minor < 0 ? "red" : "teal"}
                  style={{ fontVariantNumeric: "tabular-nums" }}
                >
                  {formatMoney(tx.amount_minor, currency, digitsOf(currency))}
                </Text>
                <ConfirmAction
                  label={t("money.tx.trash")}
                  question={t("money.tx.trashConfirm")}
                  confirmLabel={t("money.tx.trashAction")}
                  busy={remove.isPending}
                  onConfirm={() => {
                    remove.mutate(tx.id);
                  }}
                />
              </Stack>
            </Group>
          </Paper>
        );
      })}
      {transactions.hasNextPage && (
        <Group>
          <Button
            variant="default"
            onClick={() => void transactions.fetchNextPage()}
            loading={transactions.isFetchingNextPage}
          >
            {t("money.tx.loadMore")}
          </Button>
        </Group>
      )}
      <RecordDrawer
        opened={recording}
        onClose={() => {
          setRecording(false);
        }}
        accounts={accounts.data ?? []}
        categories={categories.data ?? []}
        digitsOf={digitsOf}
        base={base}
      />
    </Stack>
  );
}

type AccountOption = { id: string; name: string; currency: string };
type CategoryOption = {
  id: string;
  name: string;
  kind: "expense" | "income";
  parent_id?: string | undefined;
};

function RecordDrawer({
  opened,
  onClose,
  accounts,
  categories,
  digitsOf,
  base,
}: {
  opened: boolean;
  onClose: () => void;
  accounts: AccountOption[];
  categories: CategoryOption[];
  digitsOf: DigitsOf;
  base: string;
}) {
  const { t } = useTranslation();
  const record = useRecordTransaction();
  const [kind, setKind] = useState<Kind>("expense");
  // Until the user picks one, the first account is used, even if accounts loaded after this mounted.
  const [chosenAccount, setAccountId] = useState<string | null>(null);
  const accountId = chosenAccount ?? accounts[0]?.id ?? null;
  const [toAccountId, setToAccountId] = useState<string | null>(null);
  const [amount, setAmount] = useState("");
  const [occurredOn, setOccurredOn] = useState(todayText());
  const [categoryId, setCategoryId] = useState<string | null>(null);
  const [payee, setPayee] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);

  const account = accounts.find((a) => a.id === accountId);
  const currency = account?.currency ?? base;
  const digits = digitsOf(currency);
  const usableCategories = categories.filter((c) =>
    kind === "income" ? c.kind === "income" : c.kind === "expense",
  );
  const sameCurrencyTargets = accounts.filter((a) => a.id !== accountId && a.currency === currency);

  async function handleSubmit(e: SyntheticEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    const minor = toMinor(amount, digits);
    if (minor === null) {
      setError(t("money.amountInvalid"));
      return;
    }
    if (!accountId) {
      setError(t("money.tx.chooseAccount"));
      return;
    }
    if (kind === "transfer" && !toAccountId) {
      setError(t("money.tx.chooseTarget"));
      return;
    }
    const input: TransactionInput = {
      kind,
      account_id: accountId,
      amount_minor: minor,
      occurred_on: occurredOn,
    };
    if (kind === "transfer" && toAccountId) input.to_account_id = toAccountId;
    if (kind !== "transfer" && categoryId) input.category_id = categoryId;
    if (payee.trim()) input.payee = payee.trim();
    if (notes.trim()) input.notes = notes.trim();
    try {
      await record.mutateAsync(input);
      setAmount("");
      setPayee("");
      setNotes("");
      onClose();
    } catch (err) {
      setError(
        err instanceof ProblemError && err.problem.detail
          ? err.problem.detail
          : t("money.saveFailed"),
      );
    }
  }

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size="md"
      title={t("money.tx.record")}
    >
      <form onSubmit={(e) => void handleSubmit(e)} noValidate>
        <Stack gap="sm">
          {error && (
            <Alert color="red" role="alert">
              {error}
            </Alert>
          )}
          <SegmentedControl
            value={kind}
            onChange={(v) => {
              setKind(v);
              setCategoryId(null);
              setToAccountId(null);
            }}
            data={[
              { value: "expense", label: t("money.tx.kind.expense") },
              { value: "income", label: t("money.tx.kind.income") },
              { value: "transfer", label: t("money.tx.kind.transfer") },
            ]}
          />
          <Select
            id="tx-account"
            label={kind === "income" ? t("money.tx.toAccount") : t("money.tx.fromAccount")}
            data={accounts.map((a) => ({ value: a.id, label: `${a.name} (${a.currency})` }))}
            value={accountId}
            onChange={setAccountId}
            allowDeselect={false}
          />
          {kind === "transfer" && (
            <Select
              id="tx-to-account"
              label={t("money.tx.toAccountTransfer")}
              description={
                sameCurrencyTargets.length === 0
                  ? t("money.tx.noSameCurrency", { currency })
                  : undefined
              }
              data={sameCurrencyTargets.map((a) => ({ value: a.id, label: a.name }))}
              value={toAccountId}
              onChange={setToAccountId}
            />
          )}
          <TextInput
            id="tx-amount"
            label={t("money.tx.amount", { currency })}
            description={digits === 0 ? undefined : t("money.tx.decimals", { count: digits })}
            value={amount}
            onChange={(e) => {
              setAmount(e.currentTarget.value);
            }}
            inputMode="decimal"
          />
          <TextInput
            id="tx-date"
            type="date"
            label={t("money.tx.date")}
            value={occurredOn}
            onChange={(e) => {
              setOccurredOn(e.currentTarget.value);
            }}
          />
          {kind !== "transfer" && (
            <Select
              id="tx-category"
              label={t("money.tx.category")}
              data={usableCategories.map((c) => ({ value: c.id, label: c.name }))}
              value={categoryId}
              onChange={setCategoryId}
              clearable
            />
          )}
          <TextInput
            id="tx-payee"
            label={t("money.tx.payee")}
            value={payee}
            onChange={(e) => {
              setPayee(e.currentTarget.value);
            }}
          />
          <Textarea
            id="tx-notes"
            label={t("money.tx.notes")}
            minRows={2}
            value={notes}
            onChange={(e) => {
              setNotes(e.currentTarget.value);
            }}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={record.isPending}>
              {t("common.save")}
            </Button>
          </Group>
        </Stack>
      </form>
    </Drawer>
  );
}
