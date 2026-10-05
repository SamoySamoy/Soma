import createClient from "openapi-fetch";

import type { paths } from "./schema.gen";

/**
 * The typed API client, generated from api/openapi.yaml. All server calls go
 * through it (CLAUDE.md). fetch is looked up on each call so tests can stub it.
 */
export const api = createClient<paths>({
  baseUrl: window.location.origin,
  credentials: "same-origin",
  fetch: (input) => globalThis.fetch(input),
});
