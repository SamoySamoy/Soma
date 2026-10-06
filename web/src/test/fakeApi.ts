import { vi } from "vitest";

export type Call = { method: string; url: string; body: unknown; ifMatch: string | null };

type Options = {
  contacts?: unknown[];
  journal?: unknown[];
  tasks?: unknown[];
  self?: Record<string, unknown>;
  bodymap?: unknown;
  /** When set, every body map request fails with this status. */
  bodymapStatus?: number;
  /** Response for POST, PATCH and DELETE. */
  write?: { status: number; body?: unknown };
};

const built = new Set(["heart", "mind", "responsibilities", "self"]);

/**
 * A body map in which Heart, Mind, Responsibilities and Self are built, with
 * one birthday coming up and a journal entry from today. The rest are not built.
 */
export function defaultBodyMap() {
  const areas = [
    ["mind", "P1", { entries: 1, days_since_last: 0 }],
    ["self", "P1", { filled: 1, fields: 4 }],
    ["responsibilities", "P1", { overdue: 0, due_today: 0, overdue_long: 0 }],
    ["heart", "P1", { birthdays_soon: 1 }],
    ["body", "P2", {}],
    ["work", "P2", {}],
    ["money", "P1", {}],
    ["growth", "P2", {}],
    ["journeys", "P3", {}],
    ["home", "P2", {}],
    ["papers", "P2", {}],
  ].map(([key, phase, counts]) => {
    const enabled = built.has(key as string);
    return {
      key,
      phase,
      status: enabled ? "attention" : "unknown",
      enabled,
      counts: enabled ? counts : {},
    };
  });
  return { as_of: "2026-10-06T09:00:00Z", areas };
}

/** A Self profile with one field filled in, at version 3. */
export function defaultSelfProfile() {
  return {
    preferred_name: "Lan",
    version: 3,
    updated_at: "2026-10-05T10:00:00Z",
  };
}

/**
 * Stubs fetch with a small in-memory API and records every request. Each
 * endpoint answers separately, so one test can load several screens.
 */
export function fakeApi(options: Options = {}): Call[] {
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: Request) => {
      const url = new URL(input.url);
      const text = await input.text();
      const body: unknown = text ? JSON.parse(text) : undefined;
      calls.push({
        method: input.method,
        url: url.pathname + url.search,
        body,
        ifMatch: input.headers.get("If-Match"),
      });

      if (input.method !== "GET") {
        const write = options.write ?? { status: 201, body: undefined };
        return json(write.status, write.body);
      }
      if (url.pathname === "/api/v1/meta") {
        return json(200, { name: "Soma", version: "0.1.0", mode: "local" });
      }
      if (url.pathname === "/api/v1/bodymap") {
        if (options.bodymapStatus) {
          return json(options.bodymapStatus, {
            type: "about:blank",
            title: "Internal Server Error",
            status: options.bodymapStatus,
            code: "internal",
          });
        }
        return json(200, options.bodymap ?? defaultBodyMap());
      }
      if (url.pathname === "/api/v1/people/contacts") {
        return json(200, { items: options.contacts ?? [] });
      }
      if (url.pathname === "/api/v1/journal/entries") {
        return json(200, { items: options.journal ?? [] });
      }
      if (url.pathname === "/api/v1/tasks") {
        return json(200, { items: options.tasks ?? [] });
      }
      if (url.pathname === "/api/v1/self") {
        const profile = options.self ?? defaultSelfProfile();
        return json(200, profile, `"${String(profile.version)}"`);
      }
      return json(404, {
        type: "about:blank",
        title: "Not Found",
        status: 404,
        code: "request.not_found",
      });
    }),
  );
  return calls;
}

export function json(status: number, body: unknown, etag?: string) {
  const headers: Record<string, string> = {
    "Content-Type": status >= 400 ? "application/problem+json" : "application/json",
  };
  if (etag) headers.ETag = etag;
  return new Response(body === undefined ? null : JSON.stringify(body), { status, headers });
}
