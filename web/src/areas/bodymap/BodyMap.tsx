import { Anchor, Badge, Group, Paper, Stack, Text } from "@mantine/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router";

import type { BodyArea } from "./api";
import { areaSummary, statusColor, statusLabel } from "./status";

type Props = { areas: BodyArea[] };

/**
 * The body map (BODY-01): a figure whose regions are the areas of life. Each
 * region opens its area. A list of the same areas sits below, for keyboard
 * and screen-reader users (BODY-06).
 */
export function BodyMap({ areas }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const byKey = new Map(areas.map((a) => [a.key, a]));
  const [selected, setSelected] = useState<string>("heart");
  const current = byKey.get(selected as BodyArea["key"]);

  function open(key: string) {
    void navigate(`/me/${key}`);
  }

  function regionProps(key: string) {
    const area = byKey.get(key as BodyArea["key"]);
    const name = t(`areas.${key}.name`);
    return {
      role: "link" as const,
      tabIndex: 0,
      "aria-label": area
        ? `${name}: ${area.enabled ? statusLabel(t, area.status) : t("bodymap.status.unknown")}`
        : name,
      "data-area": key,
      style: { cursor: "pointer", opacity: area?.enabled === false ? 0.45 : 1 },
      onClick: () => {
        open(key);
      },
      onFocus: () => {
        setSelected(key);
      },
      onMouseEnter: () => {
        setSelected(key);
      },
      onKeyDown: (e: React.KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          open(key);
        }
      },
    };
  }

  function fill(key: string): {
    fill: string;
    fillOpacity: number;
    stroke: string;
    strokeWidth: number;
  } {
    const area = byKey.get(key as BodyArea["key"]);
    const color = `var(--mantine-color-${area ? statusColor(area.status) : "gray"}-${area?.enabled ? "6" : "4"})`;
    const isSelected = selected === key;
    return {
      fill: color,
      fillOpacity: isSelected ? 0.45 : 0.2,
      stroke: color,
      strokeWidth: isSelected ? 3 : 1.5,
    };
  }

  return (
    <Stack gap="md">
      <Paper withBorder radius="md" p="xs" style={{ overflowX: "auto" }}>
        <svg
          viewBox="0 0 600 530"
          role="group"
          aria-label={t("bodymap.figureLabel")}
          style={{ display: "block", width: "100%", minWidth: 420, height: "auto" }}
        >
          {/* The figure itself: not interactive. */}
          <g fill="none" stroke="var(--mantine-color-gray-5)" strokeWidth={1.5}>
            <rect x="288" y="106" width="24" height="20" rx="4" />
            <path d="M232 132 L214 136 L188 300 L210 304 L234 176 Z" />
            <path d="M368 132 L386 136 L412 300 L390 304 L366 176 Z" />
          </g>

          <g {...regionProps("mind")}>
            <path d="M260 70 A40 40 0 0 1 340 70 Z" {...fill("mind")} />
          </g>
          <g {...regionProps("self")}>
            <path d="M260 70 A40 40 0 0 0 340 70 Z" {...fill("self")} />
          </g>
          <g {...regionProps("responsibilities")}>
            <path
              d="M232 156 Q232 124 262 124 L338 124 Q368 124 368 156 Z"
              {...fill("responsibilities")}
            />
          </g>
          <g {...regionProps("body")}>
            <path d="M232 158 L368 158 L360 300 Q300 316 240 300 Z" {...fill("body")} />
          </g>
          <g {...regionProps("heart")}>
            <path
              d="M318 210 C304 199 298 190 304 182 C309 175 318 178 318 185 C318 178 327 175 332 182 C338 190 332 199 318 210 Z"
              {...fill("heart")}
            />
          </g>
          <g {...regionProps("work")}>
            <circle cx="199" cy="318" r="16" {...fill("work")} />
          </g>
          <g {...regionProps("money")}>
            <circle cx="401" cy="318" r="16" {...fill("money")} />
          </g>
          <g {...regionProps("growth")}>
            <path d="M244 304 Q270 312 296 312 L292 478 L262 478 Z" {...fill("growth")} />
            <path d="M304 312 Q330 312 356 304 L338 478 L308 478 Z" {...fill("growth")} />
          </g>
          <g {...regionProps("journeys")}>
            <ellipse cx="272" cy="490" rx="24" ry="10" {...fill("journeys")} />
            <ellipse cx="328" cy="490" rx="24" ry="10" {...fill("journeys")} />
          </g>
          <g {...regionProps("home")}>
            <path d="M66 482 L66 452 L100 424 L134 452 L134 482 Z" {...fill("home")} />
          </g>
          <g {...regionProps("papers")}>
            <path d="M468 440 L492 440 L499 448 L534 448 L534 482 L468 482 Z" {...fill("papers")} />
          </g>

          {/* Labels, each with a dot in the area's status colour. */}
          {LABELS.map((l) => {
            const area = byKey.get(l.key as BodyArea["key"]);
            return (
              <g key={l.key} aria-hidden="true">
                <circle
                  cx={l.dot[0]}
                  cy={l.dot[1]}
                  r={4.5}
                  fill={`var(--mantine-color-${area ? statusColor(area.status) : "gray"}-6)`}
                />
                <text
                  x={l.text[0]}
                  y={l.text[1]}
                  textAnchor={l.anchor}
                  fontSize={13}
                  fontWeight={600}
                  fill={
                    area?.enabled === false
                      ? "var(--mantine-color-dimmed)"
                      : "var(--mantine-color-text)"
                  }
                >
                  {t(`areas.${l.key}.name`)}
                </text>
              </g>
            );
          })}
        </svg>
      </Paper>

      {current && (
        <Paper withBorder radius="md" p="md" aria-live="polite">
          <Group justify="space-between" align="flex-start" wrap="wrap">
            <Stack gap={4}>
              <Text fw={600}>{t(`areas.${current.key}.name`)}</Text>
              <Text size="sm" c="dimmed">
                {areaSummary(t, current)}
              </Text>
            </Stack>
            <Group gap="xs">
              <Badge color={statusColor(current.status)} variant="light">
                {current.enabled ? statusLabel(t, current.status) : t("bodymap.status.unknown")}
              </Badge>
              <Anchor component={Link} to={`/me/${current.key}`} size="sm">
                {t("bodymap.openArea", { name: t(`areas.${current.key}.name`) })}
              </Anchor>
            </Group>
          </Group>
        </Paper>
      )}

      <nav aria-label={t("bodymap.listLabel")}>
        <ul
          style={{
            listStyle: "none",
            padding: 0,
            margin: 0,
            display: "grid",
            gap: 6,
            gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))",
          }}
        >
          {areas.map((a) => (
            <li key={a.key}>
              <Anchor component={Link} to={`/me/${a.key}`} size="sm">
                {t(`areas.${a.key}.name`)}
              </Anchor>
              <Text span size="sm" c="dimmed">
                {" "}
                · {a.enabled ? statusLabel(t, a.status) : t("bodymap.status.unknown")}
              </Text>
            </li>
          ))}
        </ul>
      </nav>
    </Stack>
  );
}

/** Label positions from the body map design (business spec, section 4). */
const LABELS: {
  key: string;
  dot: [number, number];
  text: [number, number];
  anchor: "end" | "start";
}[] = [
  { key: "mind", dot: [188, 42], text: [176, 47], anchor: "end" },
  { key: "responsibilities", dot: [188, 131], text: [176, 136], anchor: "end" },
  { key: "work", dot: [160, 313], text: [148, 318], anchor: "end" },
  { key: "growth", dot: [188, 393], text: [176, 398], anchor: "end" },
  { key: "self", dot: [424, 87], text: [436, 92], anchor: "start" },
  { key: "heart", dot: [424, 183], text: [436, 188], anchor: "start" },
  { key: "body", dot: [424, 249], text: [436, 254], anchor: "start" },
  { key: "money", dot: [440, 313], text: [452, 318], anchor: "start" },
  { key: "journeys", dot: [424, 405], text: [436, 410], anchor: "start" },
  { key: "home", dot: [60, 505], text: [72, 510], anchor: "start" },
  { key: "papers", dot: [468, 505], text: [480, 510], anchor: "start" },
];
