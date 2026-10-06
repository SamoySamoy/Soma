/**
 * Money amounts are integers in minor units (ADR-006). These helpers convert
 * between what people type ("125.50") and that integer, without floating point.
 */

const decimal = /^\d+(\.\d+)?$/;

/**
 * Parses what the user typed into minor units. Returns null when the text is
 * not a positive amount with at most `digits` decimal places.
 */
export function toMinor(text: string, digits: number): number | null {
  const trimmed = text.trim().replace(/,/g, "");
  if (!decimal.test(trimmed)) return null;
  const [whole = "0", fraction = ""] = trimmed.split(".");
  if (fraction.length > digits) return null;
  const padded = fraction.padEnd(digits, "0");
  const minor = Number(`${whole}${padded}`);
  return Number.isSafeInteger(minor) && minor > 0 ? minor : null;
}

/** Formats minor units for display, with the currency's digits and grouping. */
export function formatMoney(minor: number, currency: string, digits: number): string {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(minor / 10 ** digits);
}

/** Minor units as text for an input, such as 125050 with 2 digits -> "1250.50". */
export function fromMinor(minor: number, digits: number): string {
  const negative = minor < 0;
  const abs = Math.abs(minor)
    .toString()
    .padStart(digits + 1, "0");
  const whole = digits === 0 ? abs : abs.slice(0, -digits);
  const fraction = digits === 0 ? "" : abs.slice(-digits);
  return `${negative ? "-" : ""}${whole}${fraction ? `.${fraction}` : ""}`;
}

/** The first day of the month that contains `day`, as YYYY-MM-DD. */
export function monthOf(day: Date): string {
  return `${String(day.getFullYear())}-${String(day.getMonth() + 1).padStart(2, "0")}-01`;
}

export function todayText(): string {
  const d = new Date();
  return `${String(d.getFullYear())}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
