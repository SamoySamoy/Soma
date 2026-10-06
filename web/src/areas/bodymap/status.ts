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
  const c = area.counts;
  switch (area.key) {
    case "heart": {
      const soon = c["birthdays_soon"] ?? 0;
      return soon > 0
        ? t("bodymap.summary.birthdaysSoon", { count: soon })
        : t("bodymap.summary.none");
    }
    case "mind": {
      const days = c["days_since_last"] ?? 0;
      if ((c["entries"] ?? 0) === 0) return t("bodymap.summary.journalNone");
      return days > 3
        ? t("bodymap.summary.journalStale", { count: days })
        : t("bodymap.summary.none");
    }
    case "responsibilities": {
      const overdue = c["overdue"] ?? 0;
      const today = c["due_today"] ?? 0;
      if (overdue > 0) return t("bodymap.summary.overdue", { count: overdue });
      if (today > 0) return t("bodymap.summary.dueToday", { count: today });
      return t("bodymap.summary.none");
    }
    case "self": {
      const filled = c["filled"] ?? 0;
      const fields = c["fields"] ?? 0;
      return filled < fields
        ? t("bodymap.summary.selfPartial", { filled, fields })
        : t("bodymap.summary.none");
    }
    default:
      return t("bodymap.summary.none");
  }
}
