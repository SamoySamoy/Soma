import {
  Alert,
  Badge,
  Button,
  Drawer,
  Group,
  Loader,
  Paper,
  Select,
  SimpleGrid,
  Stack,
  Tabs,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { useMemo, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ConfirmAction } from "../../components/ConfirmAction";
import { ProblemError } from "../heart/api";
import {
  type Account,
  type MoneySettings,
  useAccounts,
  useCreateAccount,
  useCurrencies,
  useDeleteAccount,
  useMoneySettings,
  useMoneySummary,
  useRates,
  useSetRate,
  useUpdateMoneySettings,
} from "./api";
import { BudgetsTab } from "./BudgetsTab";
import { TransactionsTab } from "./TransactionsTab";
import { formatMoney, monthOf, toMinor, todayText } from "./money";

/** Money area (FIN-01..08): accounts, transactions, budgets, net worth and rates. */
export function MoneyPage() {
  const { t } = useTranslation();
  const settings = useMoneySettings();
  const currencies = useCurrencies();
  const digitsOf = useMemo(() => {
    const map = new Map((currencies.data ?? []).map((c) => [c.code, c.digits]));
    return (code: string) => map.get(code) ?? 2;
  }, [currencies.data]);

  if (settings.isPending || currencies.isPending) {
    return <Loader size="sm" aria-label={t("money.loading")} />;
  }
  if (settings.isError || currencies.isError) {
    return (
      <Alert color="red" role="alert">
        {t("money.loadFailed")}
      </Alert>
    );
  }

  const base = settings.data.base_currency;
  return (
    <Stack gap="md">
      <div>
        <Title order={2}>{t("money.title")}</Title>
        <Text c="dimmed">{t("money.intro", { currency: base })}</Text>
      </div>
      <Tabs defaultValue="summary" keepMounted={false}>
        <Tabs.List>
          <Tabs.Tab value="summary">{t("money.tabs.summary")}</Tabs.Tab>
          <Tabs.Tab value="transactions">{t("money.tabs.transactions")}</Tabs.Tab>
          <Tabs.Tab value="budgets">{t("money.tabs.budgets")}</Tabs.Tab>
          <Tabs.Tab value="accounts">{t("money.tabs.accounts")}</Tabs.Tab>
          <Tabs.Tab value="settings">{t("money.tabs.settings")}</Tabs.Tab>
        </Tabs.List>
        <Tabs.Panel value="summary" pt="md">
          <SummaryTab base={base} digitsOf={digitsOf} />
        </Tabs.Panel>
        <Tabs.Panel value="transactions" pt="md">
          <TransactionsTab base={base} digitsOf={digitsOf} />
        </Tabs.Panel>
        <Tabs.Panel value="budgets" pt="md">
          <BudgetsTab base={base} digitsOf={digitsOf} />
        </Tabs.Panel>
        <Tabs.Panel value="accounts" pt="md">
          <AccountsTab digitsOf={digitsOf} />
        </Tabs.Panel>
        <Tabs.Panel value="settings" pt="md">
          <SettingsTab settings={settings.data} currencies={currencies.data} digitsOf={digitsOf} />
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}

type DigitsOf = (code: string) => number;

function SummaryTab({ base, digitsOf }: { base: string; digitsOf: DigitsOf }) {
  const { t } = useTranslation();
  const [month] = useState(() => monthOf(new Date()));
  const summary = useMoneySummary(month);

  if (summary.isPending) return <Loader size="sm" aria-label={t("money.loading")} />;
  if (summary.isError) {
    return (
      <Alert color="red" role="alert">
        {t("money.loadFailed")}
      </Alert>
    );
  }
  const s = summary.data;
  const fmt = (minor: number) => formatMoney(minor, base, digitsOf(base));
  return (
    <Stack gap="md">
      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
        <Figure label={t("money.summary.netWorth")} value={fmt(s.net_worth_minor)} />
        <Figure label={t("money.summary.income")} value={fmt(s.income_minor)} />
        <Figure label={t("money.summary.expenses")} value={fmt(s.expense_minor)} />
      </SimpleGrid>
      {s.overspent_categories > 0 && (
        <Alert color="yellow">
          {t("money.summary.overspent", { count: s.overspent_categories })}
        </Alert>
      )}
      {s.missing_rates.length > 0 && (
        <Alert color="yellow" role="status">
          {t("money.summary.missingRates", { currencies: s.missing_rates.join(", ") })}
        </Alert>
      )}
      <Stack gap="xs">
        <Text fw={600}>{t("money.summary.accounts")}</Text>
        {s.accounts.length === 0 && <Text c="dimmed">{t("money.summary.noAccounts")}</Text>}
        {s.accounts.map((a) => (
          <Paper key={a.id} withBorder radius="md" p="sm">
            <Group justify="space-between" wrap="nowrap">
              <Text>{a.name}</Text>
              <Group gap="xs">
                <Text style={{ fontVariantNumeric: "tabular-nums" }}>
                  {formatMoney(a.balance_minor, a.currency, digitsOf(a.currency))}
                </Text>
                {a.base_minor !== undefined && a.currency !== base && (
                  <Badge variant="light">{fmt(a.base_minor)}</Badge>
                )}
              </Group>
            </Group>
          </Paper>
        ))}
      </Stack>
    </Stack>
  );
}

function Figure({ label, value }: { label: string; value: string }) {
  return (
    <Paper withBorder radius="md" p="md">
      <Text size="sm" c="dimmed">
        {label}
      </Text>
      <Text size="xl" fw={600} style={{ fontVariantNumeric: "tabular-nums" }}>
        {value}
      </Text>
    </Paper>
  );
}

function AccountsTab({ digitsOf }: { digitsOf: DigitsOf }) {
  const { t } = useTranslation();
  const accounts = useAccounts();
  const remove = useDeleteAccount();
  const [adding, setAdding] = useState(false);

  return (
    <Stack gap="md">
      <Group justify="flex-end">
        <Button
          onClick={() => {
            setAdding(true);
          }}
        >
          {t("money.accounts.add")}
        </Button>
      </Group>
      {accounts.isPending && <Loader size="sm" aria-label={t("money.loading")} />}
      {accounts.isSuccess && accounts.data.length === 0 && (
        <Text c="dimmed">{t("money.accounts.empty")}</Text>
      )}
      {(accounts.data ?? []).map((a: Account) => (
        <Paper key={a.id} withBorder radius="md" p="sm">
          <Group justify="space-between" wrap="nowrap">
            <Stack gap={2}>
              <Text fw={500}>{a.name}</Text>
              <Text size="sm" c="dimmed">
                {t(`money.kinds.${a.kind}`)} · {a.currency}
              </Text>
            </Stack>
            <Group gap="sm" wrap="nowrap">
              <Text style={{ fontVariantNumeric: "tabular-nums" }}>
                {formatMoney(a.balance_minor, a.currency, digitsOf(a.currency))}
              </Text>
              <ConfirmAction
                label={t("money.accounts.trash")}
                question={t("money.accounts.trashConfirm", { name: a.name })}
                confirmLabel={t("money.accounts.trashAction")}
                busy={remove.isPending}
                onConfirm={() => {
                  remove.mutate(a.id);
                }}
              />
            </Group>
          </Group>
        </Paper>
      ))}
      <AddAccountDrawer
        opened={adding}
        onClose={() => {
          setAdding(false);
        }}
        digitsOf={digitsOf}
      />
    </Stack>
  );
}

function AddAccountDrawer({
  opened,
  onClose,
  digitsOf,
}: {
  opened: boolean;
  onClose: () => void;
  digitsOf: DigitsOf;
}) {
  const { t } = useTranslation();
  const currencies = useCurrencies();
  const create = useCreateAccount();
  const [name, setName] = useState("");
  const [kind, setKind] = useState<string>("bank");
  const [currency, setCurrency] = useState<string>("VND");
  const [opening, setOpening] = useState("0");
  const [opened_on, setOpenedOn] = useState(todayText());
  const [error, setError] = useState<string | null>(null);

  const digits = digitsOf(currency);
  const openingMinor = opening.trim().startsWith("-")
    ? null
    : toMinor(opening === "" ? "0" : opening, digits);

  async function handleSubmit(e: SyntheticEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    if (name.trim() === "") {
      setError(t("money.accounts.nameRequired"));
      return;
    }
    if (openingMinor === null) {
      setError(t("money.amountInvalid"));
      return;
    }
    const negative = opening.trim().startsWith("-");
    try {
      await create.mutateAsync({
        name: name.trim(),
        kind: kind as Account["kind"],
        currency,
        opened_on,
        opening_minor: negative ? -openingMinor : openingMinor,
      });
      setName("");
      setOpening("0");
      onClose();
    } catch (err) {
      setError(
        err instanceof ProblemError
          ? (err.problem.detail ?? t("money.saveFailed"))
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
      title={t("money.accounts.add")}
    >
      <form onSubmit={(e) => void handleSubmit(e)} noValidate>
        <Stack gap="sm">
          {error && (
            <Alert color="red" role="alert">
              {error}
            </Alert>
          )}
          <TextInput
            id="account-name"
            label={t("money.accounts.name")}
            value={name}
            onChange={(e) => {
              setName(e.currentTarget.value);
            }}
            required
          />
          <Select
            id="account-kind"
            label={t("money.accounts.kind")}
            data={["cash", "bank", "credit_card", "e_wallet", "loan", "investment"].map((k) => ({
              value: k,
              label: t(`money.kinds.${k}`),
            }))}
            value={kind}
            onChange={(v) => {
              setKind(v ?? "bank");
            }}
            allowDeselect={false}
          />
          <Select
            id="account-currency"
            label={t("money.accounts.currency")}
            data={(currencies.data ?? []).map((c) => ({ value: c.code, label: c.code }))}
            value={currency}
            onChange={(v) => {
              setCurrency(v ?? "VND");
            }}
            allowDeselect={false}
          />
          <TextInput
            id="account-opening"
            label={t("money.accounts.opening")}
            description={t("money.accounts.openingHint")}
            value={opening}
            onChange={(e) => {
              setOpening(e.currentTarget.value);
            }}
          />
          <TextInput
            id="account-opened"
            type="date"
            label={t("money.accounts.openedOn")}
            value={opened_on}
            onChange={(e) => {
              setOpenedOn(e.currentTarget.value);
            }}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={create.isPending}>
              {t("common.save")}
            </Button>
          </Group>
        </Stack>
      </form>
    </Drawer>
  );
}

function SettingsTab({
  settings,
  currencies,
  digitsOf,
}: {
  settings: MoneySettings;
  currencies: { code: string; digits: number }[];
  digitsOf: DigitsOf;
}) {
  const { t } = useTranslation();
  const update = useUpdateMoneySettings();
  const [message, setMessage] = useState<string | null>(null);
  return (
    <Stack gap="xl">
      <Stack gap="sm" maw={420}>
        <Text fw={600}>{t("money.settings.base")}</Text>
        <Text size="sm" c="dimmed">
          {t("money.settings.baseHint")}
        </Text>
        <Select
          id="base-currency"
          label={t("money.settings.baseLabel")}
          data={currencies.map((c) => ({ value: c.code, label: c.code }))}
          value={settings.base_currency}
          onChange={(v) => {
            if (!v || v === settings.base_currency) return;
            setMessage(null);
            update.mutate(
              { version: settings.version, patch: { base_currency: v } },
              {
                onSuccess: () => {
                  setMessage(t("money.settings.saved"));
                },
                onError: (err) => {
                  setMessage(
                    err instanceof ProblemError && err.problem.status === 412
                      ? t("money.stale")
                      : t("money.saveFailed"),
                  );
                },
              },
            );
          }}
          allowDeselect={false}
        />
        {message && (
          <Text size="sm" role="status">
            {message}
          </Text>
        )}
      </Stack>
      <RatesSection base={settings.base_currency} currencies={currencies} digitsOf={digitsOf} />
    </Stack>
  );
}

function RatesSection({
  base,
  currencies,
}: {
  base: string;
  currencies: { code: string; digits: number }[];
  digitsOf: DigitsOf;
}) {
  const { t } = useTranslation();
  const rates = useRates();
  const setRate = useSetRate();
  const foreign = currencies.filter((c) => c.code !== base);
  const [currency, setCurrency] = useState<string | null>(null);
  const [rateText, setRateText] = useState("");
  const [rateOn, setRateOn] = useState(todayText());
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: SyntheticEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    if (!currency) {
      setError(t("money.rates.chooseCurrency"));
      return;
    }
    // A rate has eight decimal places: rate_e8 is the rate times 10^8.
    const rateE8 = toMinor(rateText, 8);
    if (rateE8 === null) {
      setError(t("money.rates.invalid"));
      return;
    }
    try {
      await setRate.mutateAsync({ currency, rate_on: rateOn, rate_e8: rateE8 });
      setRateText("");
    } catch {
      setError(t("money.saveFailed"));
    }
  }

  return (
    <Stack gap="sm">
      <Text fw={600}>{t("money.rates.title")}</Text>
      <Text size="sm" c="dimmed">
        {t("money.rates.hint", { base })}
      </Text>
      <form onSubmit={(e) => void handleSubmit(e)} noValidate>
        <Stack gap="sm" maw={420}>
          {error && (
            <Alert color="red" role="alert">
              {error}
            </Alert>
          )}
          <Select
            id="rate-currency"
            label={t("money.rates.currency")}
            data={foreign.map((c) => ({ value: c.code, label: c.code }))}
            value={currency}
            onChange={setCurrency}
          />
          <TextInput
            id="rate-value"
            label={t("money.rates.value", { base })}
            value={rateText}
            onChange={(e) => {
              setRateText(e.currentTarget.value);
            }}
          />
          <TextInput
            id="rate-on"
            type="date"
            label={t("money.rates.on")}
            value={rateOn}
            onChange={(e) => {
              setRateOn(e.currentTarget.value);
            }}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={setRate.isPending}>
              {t("money.rates.save")}
            </Button>
          </Group>
        </Stack>
      </form>
      {(rates.data ?? []).map((r) => (
        <Text key={r.currency} size="sm" style={{ fontVariantNumeric: "tabular-nums" }}>
          {t("money.rates.line", {
            currency: r.currency,
            rate: r.rate_e8 / 1e8,
            base,
            on: r.rate_on,
          })}
        </Text>
      ))}
    </Stack>
  );
}
