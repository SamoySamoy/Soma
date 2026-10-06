import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import { fakeApi } from "./test/fakeApi";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Landing page", () => {
  it("shows the body map and the instance version", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    expect(await screen.findByRole("heading", { name: "Your body map" })).toBeInTheDocument();
    expect(await screen.findByText("Version 0.1.0, local mode")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Heart: Needs attention" })).toBeInTheDocument();
  });

  it("explains when the server cannot be reached", async () => {
    fakeApi({ bodymapStatus: 500 });
    render(<App initialPath="/" />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Can't reach the Soma server");
  });
});

describe("Body map", () => {
  it("lists every area in the text alternative, with its status", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    const list = await screen.findByRole("navigation", { name: "All areas" });
    expect(within(list).getAllByRole("link")).toHaveLength(11);
    expect(within(list).getByRole("link", { name: "Money" })).toBeInTheDocument();
  });

  it("opens the area when its region is clicked", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    const user = userEvent.setup();
    await user.click(await screen.findByRole("link", { name: "Heart: Needs attention" }));

    expect(await screen.findByRole("heading", { name: "People" })).toBeInTheDocument();
  });

  it("opens the area from the keyboard with Enter", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    const region = await screen.findByRole("link", { name: "Heart: Needs attention" });
    region.focus();
    const user = userEvent.setup();
    await user.keyboard("{Enter}");

    expect(await screen.findByRole("heading", { name: "People" })).toBeInTheDocument();
  });

  it("describes a coming-soon area and says when it arrives", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    const user = userEvent.setup();
    await user.click(await screen.findByRole("link", { name: "Money: Not available yet" }));

    expect(await screen.findByText("Not available yet. It arrives in P1.")).toBeInTheDocument();
  });

  it("summarises the selected area from its counts", async () => {
    fakeApi();
    render(<App initialPath="/" />);

    expect(await screen.findByText("1 birthday in the next 14 days.")).toBeInTheDocument();
  });
});
