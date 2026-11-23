/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useId } from "react";
import { observer } from "mobx-react";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { INTAKE_DATE_PRESETS, INTAKE_STATUS_LABELS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedDate } from "@nerve/utils";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
// store
import type { DayRange, IntakeFilters } from "@/store/issue/intake/filters";
import { customized, defaultFilters, presetOf, toggledStatuses } from "@/store/issue/intake/filters";
// local imports
import { useIntakePageState } from "../page-state";

/** A value of a filter applied: its name, and the filters without it, from those held then; none for one that stays. */
type Chip = { key: string; label: string; remove?: (current: IntakeFilters) => IntakeFilters };

/** The page's translation. */
type Translate = ReturnType<typeof useTranslation>["t"];

/** A day as the page shows it. */
const shownDay = (day: string) => renderFormattedDate(day) ?? day;

/** A range of days by its preset at now, if one names it; else after a day, before a day, or a day or two. */
function rangeLabel(t: Translate, range: DayRange, now: Date): string {
  const preset = INTAKE_DATE_PRESETS.find((option) => option.value === presetOf(range, now));
  if (preset) return t(preset.i18n_label);
  if (range.from !== undefined && range.to !== undefined)
    return range.from === range.to
      ? shownDay(range.from)
      : t("inbox_issue.filters.between", { from: shownDay(range.from), to: shownDay(range.to) });
  if (range.from !== undefined) return t("inbox_issue.filters.after", { date: shownDay(range.from) });
  return t("inbox_issue.filters.before", { date: shownDay(range.to ?? "") });
}

/**
 * The filters applied to the list of the intake page's tab (P5a spec 2.7): nothing while they are the tab's first
 * ones, the order aside (customized); else each filter, by its name, with each of its values by its name, and a button
 * that removes that value alone, named as removing it, never submitting a form around it; the last status has none,
 * as a list names a status always (R9). "Clear all" goes back to the tab's first filters, the order kept.
 */
export const IntakeAppliedFilters = observer(function IntakeAppliedFilters({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const id = useId();
  const { filters, setFilters } = useIntakePageState();
  const { getUserDetails } = useMember();
  const { getProjectLabels } = useLabel();
  if (!customized(filters)) return null;

  const now = new Date();
  const unknown = t("common.filter_options.unavailable");
  const labels = getProjectLabels(projectId) ?? [];
  const without =
    (field: "priorities" | "assignees" | "creators" | "labels", value: string) => (current: IntakeFilters) => ({
      ...current,
      [field]: current[field].filter((held) => held !== value),
    });
  const members = (field: "assignees" | "creators"): Chip[] =>
    filters[field].map((member) => ({
      key: member,
      label: getUserDetails(member)?.display_name ?? unknown,
      remove: without(field, member),
    }));
  const days = (field: "created" | "updated"): Chip[] => {
    const range = filters[field];
    return range === undefined
      ? []
      : [{ key: field, label: rangeLabel(t, range, now), remove: (current) => ({ ...current, [field]: undefined }) }];
  };
  const groups: { key: string; title: string; chips: Chip[] }[] = [
    {
      key: "statuses",
      title: t("inbox_issue.filters.status"),
      chips: filters.statuses.map((status) => ({
        key: status,
        label: t(INTAKE_STATUS_LABELS[status]),
        remove:
          toggledStatuses(filters.statuses, status) === undefined
            ? undefined
            : (current) => {
                const statuses = toggledStatuses(current.statuses, status);
                return statuses === undefined ? current : { ...current, statuses };
              },
      })),
    },
    {
      key: "priorities",
      title: t("common.priority"),
      chips: filters.priorities.map((priority) => ({
        key: priority,
        label: t(priority),
        remove: without("priorities", priority),
      })),
    },
    { key: "assignees", title: t("common.assignees"), chips: members("assignees") },
    { key: "creators", title: t("common.created_by"), chips: members("creators") },
    {
      key: "labels",
      title: t("common.labels"),
      chips: filters.labels.map((label) => ({
        key: label,
        label: labels.find((held) => held.id === label)?.name ?? unknown,
        remove: without("labels", label),
      })),
    },
    { key: "created", title: t("inbox_issue.filters.created_at"), chips: days("created") },
    { key: "updated", title: t("inbox_issue.filters.updated_at"), chips: days("updated") },
  ];

  return (
    <div className="flex flex-shrink-0 flex-wrap items-center gap-2 border-b border-subtle px-3 py-2">
      {groups
        .filter((group) => group.chips.length > 0)
        .map((group) => (
          <div
            key={group.key}
            role="group"
            aria-labelledby={`${id}-${group.key}`}
            className="flex flex-wrap items-center gap-1 rounded-md border border-subtle p-1 text-11"
          >
            <span id={`${id}-${group.key}`} className="px-0.5 text-tertiary">
              {group.title}
            </span>
            {group.chips.map((chip) => (
              <span key={chip.key} className="flex items-center gap-1 rounded-sm bg-layer-3 px-1.5 py-0.5">
                <span className="text-secondary">{chip.label}</span>
                {chip.remove && (
                  <button
                    type="button"
                    className="grid place-items-center text-tertiary hover:text-secondary"
                    aria-label={t("common.remove_item", { name: chip.label })}
                    onClick={() => setFilters(chip.remove!)}
                  >
                    <CloseOutline height={10} width={10} aria-hidden="true" />
                  </button>
                )}
              </span>
            ))}
          </div>
        ))}
      <PillButton
        type="button"
        size="md"
        variant="outline"
        label={t("common.clear_all")}
        endIcon={<CloseOutline height={12} width={12} aria-hidden="true" />}
        onClick={() => setFilters((current) => ({ ...defaultFilters(current.tab), order: current.order }))}
      />
    </div>
  );
});
