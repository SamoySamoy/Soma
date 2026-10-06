import { AppShell, Anchor, Group, MantineProvider, Text, Title, createTheme } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Link,
  Outlet,
  RouterProvider,
  createBrowserRouter,
  createMemoryRouter,
} from "react-router";

import { HomePage } from "./areas/home/HomePage";
import { AreaPage } from "./areas/AreaPage";

const theme = createTheme({
  primaryColor: "teal",
  fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
});

function Layout() {
  const { t } = useTranslation();
  return (
    <AppShell header={{ height: 56 }} padding="md">
      <AppShell.Header>
        <Group h="100%" px="md" gap="lg" wrap="nowrap">
          <Title order={2} size="h4">
            {t("app.name")}
          </Title>
          <Group gap="md" component="nav" aria-label="Main">
            <Anchor component={Link} to="/" size="sm">
              {t("nav.home")}
            </Anchor>
            <Anchor component={Link} to="/me/heart" size="sm">
              {t("nav.heart")}
            </Anchor>
          </Group>
          <Text size="sm" c="dimmed" visibleFrom="md">
            {t("app.tagline")}
          </Text>
        </Group>
      </AppShell.Header>
      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}

const routes = [
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: "me/:area", element: <AreaPage /> },
    ],
  },
];

type AppProps = {
  /** Start at this path with an in-memory router (tests). */
  initialPath?: string;
};

export function App({ initialPath }: AppProps) {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: initialPath ? false : 2 } } }),
  );
  const [router] = useState(() =>
    initialPath
      ? createMemoryRouter(routes, { initialEntries: [initialPath] })
      : createBrowserRouter(routes),
  );

  return (
    <MantineProvider theme={theme} defaultColorScheme="auto">
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </MantineProvider>
  );
}
