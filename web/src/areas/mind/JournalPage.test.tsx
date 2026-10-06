import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "../../App";
import { type Call, fakeApi } from "../../test/fakeApi";

const entry = {
  id: "0195f3a2-0000-7000-8000-00000000000a",
  entry_date: "2026-10-05",
  title: "Slow day",
  body: "A quiet walk by the lake.",
  mood: 2,
  created_at: "2026-10-05T20:00:00Z",
  updated_at: "2026-10-05T20:00:00Z",
  version: 2,
};

function callsOf(calls: Call[], method: string) {
  return calls.filter((c) => c.method === method);
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Journal page", () => {
  it("shows your entries, newest first", async () => {
    fakeApi({ journal: [entry] });
    render(<App initialPath="/me/mind" />);

    expect(await screen.findByText("Slow day")).toBeInTheDocument();
    expect(screen.getByText("A quiet walk by the lake.")).toBeInTheDocument();
    expect(screen.getByText("2026-10-05")).toBeInTheDocument();
  });

  it("refuses an empty entry", async () => {
    const calls = fakeApi();
    render(<App initialPath="/me/mind" />);
    await screen.findByText("No entries yet. Write about today, however small.");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Write an entry" }));
    const drawer = await screen.findByRole("dialog");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    expect(await within(drawer).findByRole("alert")).toHaveTextContent("Write something first.");
    expect(callsOf(calls, "POST")).toHaveLength(0);
  });

  it("saves a new entry with today's date", async () => {
    const calls = fakeApi();
    render(<App initialPath="/me/mind" />);
    await screen.findByText("No entries yet. Write about today, however small.");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Write an entry" }));
    const drawer = await screen.findByRole("dialog");
    await user.type(within(drawer).getByLabelText("What happened?"), "Went out for coffee.");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "POST")).toHaveLength(1);
    });
    const body = callsOf(calls, "POST")[0]?.body as Record<string, unknown>;
    expect(body.body).toBe("Went out for coffee.");
    expect(body.entry_date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("edits an entry with its version and clears a removed title with null", async () => {
    const calls = fakeApi({ journal: [entry] });
    render(<App initialPath="/me/mind" />);
    await screen.findByText("Slow day");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Edit the entry for 2026-10-05" }));
    const drawer = await screen.findByRole("dialog");
    await user.clear(within(drawer).getByLabelText("Title (optional)"));
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "PATCH")).toHaveLength(1);
    });
    const patch = callsOf(calls, "PATCH")[0];
    expect(patch?.url).toBe(`/api/v1/journal/entries/${entry.id}`);
    expect(patch?.ifMatch).toBe('"2"');
    expect(patch?.body).toMatchObject({ body: "A quiet walk by the lake.", title: null, mood: 2 });
  });
});
