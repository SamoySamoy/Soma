import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "../../App";
import { type Call, fakeApi } from "../../test/fakeApi";

const rent = {
  id: "0195f3a2-0000-7000-8000-00000000000b",
  title: "Pay rent",
  due_on: "2026-10-01",
  completed: false,
  overdue: true,
  created_at: "2026-09-30T10:00:00Z",
  updated_at: "2026-09-30T10:00:00Z",
  version: 1,
};

const call = {
  id: "0195f3a2-0000-7000-8000-00000000000c",
  title: "Call mum",
  completed: true,
  overdue: false,
  completed_at: "2026-10-05T09:00:00Z",
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-05T09:00:00Z",
  version: 2,
};

function callsOf(calls: Call[], method: string) {
  return calls.filter((c) => c.method === method);
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Tasks page", () => {
  it("shows open tasks and hides finished ones until asked", async () => {
    fakeApi({ tasks: [rent, call] });
    render(<App initialPath="/me/responsibilities" />);

    expect(await screen.findByText("Pay rent")).toBeInTheDocument();
    expect(screen.getByText("Overdue")).toBeInTheDocument();
    expect(screen.queryByText("Call mum")).not.toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("checkbox", { name: "Show finished tasks" }));
    expect(await screen.findByText("Call mum")).toBeInTheDocument();
  });

  it("marks a task done with its own endpoint", async () => {
    const calls = fakeApi({
      tasks: [rent],
      write: { status: 200, body: { ...rent, completed: true, version: 2 } },
    });
    render(<App initialPath="/me/responsibilities" />);
    await screen.findByText("Pay rent");

    const user = userEvent.setup();
    await user.click(screen.getByRole("checkbox", { name: "Mark Pay rent as done" }));

    await waitFor(() => {
      expect(callsOf(calls, "POST")).toHaveLength(1);
    });
    expect(callsOf(calls, "POST")[0]?.url).toBe(`/api/v1/tasks/${rent.id}/complete`);
  });

  it("will not save a task without a title", async () => {
    const calls = fakeApi();
    render(<App initialPath="/me/responsibilities" />);
    await screen.findByText("Nothing to do. Add a task when something comes up.");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Add task" }));
    const drawer = await screen.findByRole("dialog");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    expect(await within(drawer).findByRole("alert")).toHaveTextContent("Enter what needs doing.");
    expect(callsOf(calls, "POST")).toHaveLength(0);
  });
});
