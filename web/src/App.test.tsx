import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { App } from "./App";

function stubFetch(status: number, body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify(body), {
          status,
          headers: {
            "Content-Type": status < 400 ? "application/json" : "application/problem+json",
          },
        }),
      ),
    ),
  );
}

describe("App", () => {
  it("shows the instance version and mode from the API", async () => {
    stubFetch(200, { name: "Soma", version: "0.1.0", mode: "local" });
    render(<App initialPath="/" />);

    expect(screen.getByRole("heading", { name: "Welcome to Soma" })).toBeInTheDocument();
    expect(await screen.findByText("Version 0.1.0, local mode")).toBeInTheDocument();
  });

  it("explains when the server can't be reached", async () => {
    stubFetch(500, {
      type: "about:blank",
      title: "Internal Server Error",
      status: 500,
      code: "internal",
    });
    render(<App initialPath="/" />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Can't reach the Soma server");
  });
});
