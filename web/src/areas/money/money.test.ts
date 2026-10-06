import { describe, expect, it } from "vitest";

import { fromMinor, monthOf, toMinor } from "./money";

describe("money amounts", () => {
  it("reads what the user typed into minor units, without floating point", () => {
    expect(toMinor("125000", 0)).toBe(125000);
    expect(toMinor("12.50", 2)).toBe(1250);
    expect(toMinor("12", 2)).toBe(1200);
    expect(toMinor("1,250", 0)).toBe(1250);
    expect(toMinor(" 3.1 ", 2)).toBe(310);
  });

  it("refuses text that is not a positive amount with the currency's digits", () => {
    expect(toMinor("12.345", 2)).toBeNull();
    expect(toMinor("1250.5", 0)).toBeNull();
    expect(toMinor("abc", 2)).toBeNull();
    expect(toMinor("0", 2)).toBeNull();
    expect(toMinor("-5", 2)).toBeNull();
    expect(toMinor("", 2)).toBeNull();
  });

  it("turns minor units back into text for an input", () => {
    expect(fromMinor(1250, 2)).toBe("12.50");
    expect(fromMinor(5, 2)).toBe("0.05");
    expect(fromMinor(125000, 0)).toBe("125000");
    expect(fromMinor(-5, 2)).toBe("-0.05");
  });

  it("names the first day of the month", () => {
    expect(monthOf(new Date(2026, 9, 15))).toBe("2026-10-01");
  });
});
