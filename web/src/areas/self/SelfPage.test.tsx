import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "../../App";
import { type Call, fakeApi } from "../../test/fakeApi";

afterEach(() => {
  vi.unstubAllGlobals();
});

function callsOf(calls: Call[], method: string) {
  return calls.filter((c) => c.method === method);
}

describe("Self page", () => {
  it("shows the profile as it is stored", async () => {
    fakeApi({ self: { preferred_name: "Lan", version: 3, updated_at: "2026-10-05T10:00:00Z" } });
    render(<App initialPath="/me/self" />);

    expect(await screen.findByLabelText("Name you go by")).toHaveValue("Lan");
    expect(screen.getByLabelText("About you")).toHaveValue("");
  });

  it("saves with the version it loaded, and clears emptied fields with null", async () => {
    const calls = fakeApi({
      self: { preferred_name: "Lan", version: 3, updated_at: "2026-10-05T10:00:00Z" },
      write: {
        status: 200,
        body: { preferred_name: "Lan", version: 4, updated_at: "2026-10-06T10:00:00Z" },
      },
    });
    render(<App initialPath="/me/self" />);
    const name = await screen.findByLabelText("Name you go by");
    expect(name).toHaveValue("Lan");

    const user = userEvent.setup();
    await user.clear(name);
    await user.type(name, "Lana");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "PATCH")).toHaveLength(1);
    });
    const patch = callsOf(calls, "PATCH")[0];
    expect(patch?.url).toBe("/api/v1/self");
    expect(patch?.ifMatch).toBe('"3"');
    expect(patch?.body).toMatchObject({ preferred_name: "Lana", bio: null, birth_date: null });
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
  });
});
