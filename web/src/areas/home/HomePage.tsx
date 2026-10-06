import { Alert, Loader, Stack, Text, Title } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { api } from "../../api/client";
import { useBodyMap } from "../bodymap/api";
import { BodyMap } from "../bodymap/BodyMap";

/** The landing page: the body map (BODY-01). */
export function HomePage() {
  const { t } = useTranslation();
  const map = useBodyMap();
  const meta = useQuery({
    queryKey: ["meta"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/meta");
      if (error) throw new Error(error.detail ?? error.title);
      return data;
    },
  });

  return (
    <Stack gap="lg">
      <div>
        <Title order={1}>{t("home.title")}</Title>
        <Text c="dimmed">{t("home.intro")}</Text>
      </div>

      {map.isPending && <Loader size="sm" aria-label={t("bodymap.loading")} />}
      {map.isError && (
        <Alert color="red" role="alert">
          {t("home.offline")}
        </Alert>
      )}
      {map.data && <BodyMap areas={map.data.areas} />}

      {meta.data && (
        <Text size="sm" c="dimmed">
          {t("home.instance", { version: meta.data.version, mode: t(`mode.${meta.data.mode}`) })}
        </Text>
      )}
    </Stack>
  );
}
