import type { TFunction } from "i18next";

import type { BodyArea } from "./api";

/** Mantine colour for each status. Unknown (not built yet) is neutral grey. */
export function statusColor(status: BodyArea["status"]): string {
  switch (status) {
    case "calm":
      return "teal";
    case "attention":
      return "yellow";
    case "urgent":
      return "red";
    default:
      return "gray";
  }
}

export function statusLabel(t: TFunction, status: BodyArea["status"]): string {
  return t(`bodymap.status.${status}`);
}

/**
 * One line describing an area, from its counts. Disabled areas say when they
 * arrive instead, so the map never shows an empty summary.
 */
export function areaSummary(t: TFunction, area: BodyArea): string {
  if (!area.enabled) {
    return t("bodymap.notBuilt", { phase: area.phase });
  }
  const soon = area.counts["birthdays_soon"] ?? 0;
  if (area.key === "heart" && soon > 0) {
    return t("bodymap.summary.birthdaysSoon", { count: soon });
  }
  return t("bodymap.summary.none");
}
