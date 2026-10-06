import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "../../App";
import { type Call, fakeApi } from "../../test/fakeApi";

const contact = {
  id: "0195f3a2-0000-7000-8000-000000000001",
  display_name: "Lan Nguyen",
  nickname: "Lan",
  phone: "+84 90 000 0000",
  email: "lan@example.com",
  birthday: "1990-04-23",
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-01T10:00:00Z",
  version: 3,
};

function callsOf(calls: Call[], method: string) {
  return calls.filter((c) => c.method === method);
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("People page", () => {
  it("lists contacts from the API", async () => {
    fakeApi({ contacts: [contact] });
    render(<App initialPath="/me/heart" />);

    expect(await screen.findByText("Lan Nguyen")).toBeInTheDocument();
    expect(screen.getByText("(Lan)")).toBeInTheDocument();
    expect(screen.getByText("lan@example.com")).toBeInTheDocument();
  });

  it("shows an empty state that invites the first entry", async () => {
    fakeApi({ contacts: [] });
    render(<App initialPath="/me/heart" />);

    expect(
      await screen.findByText("No people yet. Add the first person you know."),
    ).toBeInTheDocument();
  });

  it("does not send a contact without a name", async () => {
    const calls = fakeApi({ contacts: [contact] });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Add person" }));
    await user.click(await screen.findByRole("button", { name: "Save" }));

    expect(await screen.findByText("Enter a name.")).toBeInTheDocument();
    expect(callsOf(calls, "POST").length > 0).toBe(false);
  });

  it("rejects a birthday that isn't a calendar date", async () => {
    const calls = fakeApi({ contacts: [contact] });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Add person" }));
    const drawer = await screen.findByRole("dialog");
    await user.type(within(drawer).getByLabelText(/^Name/), "Minh");
    await user.type(within(drawer).getByLabelText(/^Birthday/), "23/04/1990");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    expect(await within(drawer).findByText("Use the format YYYY-MM-DD.")).toBeInTheDocument();
    expect(callsOf(calls, "POST").length > 0).toBe(false);
  });

  it("creates a contact and leaves out fields that were left blank", async () => {
    const calls = fakeApi({ contacts: [contact] });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Add person" }));
    const drawer = await screen.findByRole("dialog");
    await user.type(within(drawer).getByLabelText(/^Name/), "  Minh Tran  ");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "POST").length > 0).toBe(true);
    });
    const post = calls.find((c) => c.method === "POST");
    expect(post?.url).toBe("/api/v1/people/contacts");
    expect(post?.body).toEqual({ display_name: "Minh Tran" });
  });

  it("sends the version being edited in If-Match and clears emptied fields with null", async () => {
    const calls = fakeApi({
      contacts: [contact],
      write: { status: 200, body: { ...contact, version: 4 } },
    });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Edit Lan Nguyen" }));
    const drawer = await screen.findByRole("dialog");
    await user.clear(within(drawer).getByLabelText(/^Phone/));
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(calls.some((c) => c.method === "PATCH")).toBe(true);
    });
    const patch = calls.find((c) => c.method === "PATCH");
    expect(patch?.url).toBe(`/api/v1/people/contacts/${contact.id}`);
    expect(patch?.ifMatch).toBe('"3"');
    expect(patch?.body).toMatchObject({
      display_name: "Lan Nguyen",
      phone: null,
      email: "lan@example.com",
    });
  });

  it("explains when someone else changed the contact first", async () => {
    fakeApi({
      contacts: [contact],
      write: {
        status: 412,
        body: {
          type: "about:blank",
          title: "Precondition Failed",
          status: 412,
          code: "people.stale_version",
          detail: "changed",
        },
      },
    });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Edit Lan Nguyen" }));
    const drawer = await screen.findByRole("dialog");
    await user.type(within(drawer).getByLabelText(/^Nickname/), "L");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    expect(await within(drawer).findByRole("alert")).toHaveTextContent(
      "This person was changed elsewhere",
    );
  });

  it("moves a contact to the trash only after confirming", async () => {
    const calls = fakeApi({ contacts: [contact], write: { status: 204 } });
    render(<App initialPath="/me/heart" />);
    await screen.findByText("Lan Nguyen");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Edit Lan Nguyen" }));
    const drawer = await screen.findByRole("dialog");
    await user.click(within(drawer).getByRole("button", { name: "Move to trash" }));
    expect(calls.some((c) => c.method === "DELETE")).toBe(false);

    await user.click(within(drawer).getByRole("button", { name: "Yes, move to trash" }));
    await waitFor(() => {
      expect(calls.some((c) => c.method === "DELETE")).toBe(true);
    });
    expect(calls.find((c) => c.method === "DELETE")?.url).toBe(
      `/api/v1/people/contacts/${contact.id}`,
    );
  });
});
