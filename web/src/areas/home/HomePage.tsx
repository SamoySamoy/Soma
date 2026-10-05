import { Alert, Loader, Stack, Text, Title } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { api } from "../../api/client";

export function HomePage() {
  const { t } = useTranslation();
  const meta = useQuery({
    queryKey: ["meta"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/meta");
      if (error) throw new Error(error.detail ?? error.title);
      return data;
    },
  });

  return (
    <Stack gap="md" maw={640}>
      <Title order={1}>{t("home.title")}</Title>
      <Text c="dimmed">{t("home.bodyMapComing")}</Text>
      {meta.isPending && <Loader size="sm" aria-label={t("home.loading")} />}
      {meta.isError && (
        <Alert color="red" role="alert">
          {t("home.offline")}
        </Alert>
      )}
      {meta.data && (
        <Text size="sm" c="dimmed">
          {t("home.instance", { version: meta.data.version, mode: t(`mode.${meta.data.mode}`) })}
        </Text>
      )}
    </Stack>
  );
}
