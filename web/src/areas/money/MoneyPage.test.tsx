import { render, screen, waitFor, within } from "@testing-library/react";
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

describe("Money page", () => {
  it("shows net worth and says which currency is missing a rate", async () => {
    fakeApi();
    render(<App initialPath="/me/money" />);

    expect(await screen.findByText("Net worth")).toBeInTheDocument();
    expect(screen.getByText("Income this month")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Add an exchange rate for USD");
    expect(screen.getByText("1 budget is over this month.")).toBeInTheDocument();
  });

  it("records an expense as a whole-number amount in the account's currency", async () => {
    const calls = fakeApi();
    render(<App initialPath="/me/money" />);

    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Transactions" }));
    const recordButton = await screen.findByRole("button", { name: "Record" });
    await waitFor(() => {
      expect(recordButton).toBeEnabled();
    });
    await user.click(recordButton);
    const drawer = await screen.findByRole("dialog");
    await user.type(within(drawer).getByLabelText(/^Amount in VND/), "125000");
    await user.click(within(drawer).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "POST")).toHaveLength(1);
    });
    const body = callsOf(calls, "POST")[0]?.body as Record<string, unknown>;
    expect(callsOf(calls, "POST")[0]?.url).toBe("/api/v1/money/transactions");
    expect(body).toMatchObject({
      kind: "expense",
      account_id: "0195f3a2-0000-7000-8000-000000000101",
      amount_minor: 125000,
    });
    expect(body.occurred_on).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("sets a category's budget from the amount typed", async () => {
    const calls = fakeApi();
    render(<App initialPath="/me/money" />);

    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Budgets" }));
    const field = await screen.findByLabelText("Budget for Food in VND");
    await user.clear(field);
    await user.type(field, "5000000");
    const row = field.closest("[class*='Paper']") ?? field.parentElement;
    await user.click(within(row as HTMLElement).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(callsOf(calls, "PUT")).toHaveLength(1);
    });
    expect(callsOf(calls, "PUT")[0]?.body).toMatchObject({
      category_id: "cat-food",
      amount_minor: 5000000,
    });
  });
});
