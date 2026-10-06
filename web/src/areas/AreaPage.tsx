import { Alert, Anchor, Badge, Group, Loader, Stack, Text, Title } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router";

import { type BodyArea, useBodyMap } from "./bodymap/api";
import { areaSummary, statusColor, statusLabel } from "./bodymap/status";
import { JournalPage } from "./mind/JournalPage";
import { MoneyPage } from "./money/MoneyPage";
import { PeoplePage } from "./heart/PeoplePage";
import { SelfPage } from "./self/SelfPage";
import { TasksPage } from "./responsibilities/TasksPage";

/**
 * One area of life. It opens with the area's status and summary (BODY-05),
 * then shows the area's content. Areas not built yet explain when they arrive.
 */
export function AreaPage() {
  const { t } = useTranslation();
  const { area: key } = useParams();
  const map = useBodyMap();

  if (map.isPending) {
    return <Loader size="sm" aria-label={t("bodymap.loading")} />;
  }
  if (map.isError) {
    return (
      <Alert color="red" role="alert">
        {t("bodymap.loadFailed")}
      </Alert>
    );
  }

  const area = map.data.areas.find((a) => a.key === key);
  if (!area) {
    return (
      <Stack gap="sm">
        <Title order={1}>{t("area.unknown")}</Title>
        <Anchor component={Link} to="/">
          {t("area.backToMap")}
        </Anchor>
      </Stack>
    );
  }

  return (
    <Stack gap="lg">
      <AreaHeader area={area} />
      {area.enabled && <AreaContent area={area.key} />}
    </Stack>
  );
}

/** Which page each enabled area shows. */
function AreaContent({ area }: { area: BodyArea["key"] }) {
  switch (area) {
    case "heart":
      return <PeoplePage />;
    case "mind":
      return <JournalPage />;
    case "responsibilities":
      return <TasksPage />;
    case "self":
      return <SelfPage />;
    case "money":
      return <MoneyPage />;
    default:
      return null;
  }
}

function AreaHeader({ area }: { area: BodyArea }) {
  const { t } = useTranslation();
  return (
    <Stack gap="xs">
      <Anchor component={Link} to="/" size="sm">
        {t("area.backToMap")}
      </Anchor>
      <Group gap="sm" align="center">
        <Title order={1}>{t(`areas.${area.key}.name`)}</Title>
        <Badge color={statusColor(area.status)} variant="light">
          {area.enabled ? statusLabel(t, area.status) : t("bodymap.status.unknown")}
        </Badge>
      </Group>
      <Text c="dimmed">{areaSummary(t, area)}</Text>
    </Stack>
  );
}
