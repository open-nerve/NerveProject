/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState } from "react";
import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { CloseOutline, FilterOutline, SearchOutline } from "@makeplane/propel/icons";
// nerve imports
import { INTAKE_DATE_PRESETS, INTAKE_STATUS_LABELS, ISSUE_PRIORITIES } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { PriorityIcon } from "@nerve/propel/icons";
import { toggledOptions } from "@nerve/utils";
// components
import { DateFilterModal } from "@/components/core/filters/date-filter-modal";
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { FilterHeader, FilterOption, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
// store
import type { DayRange } from "@/store/issue/intake/filters";
import {
  customRange,
  customized,
  presetOf,
  presetRange,
  tabStatuses,
  toggledStatuses,
} from "@/store/issue/intake/filters";
// local imports
import { useIntakePageState } from "../page-state";

/** An option of a filter of the menu: its name, an icon, whether it is chosen, and what choosing it does. */
type Option = { key: string; label: string; icon?: ReactNode; checked: boolean; pick: () => void };

type SectionProps = {
  /** The filter's name, in the page's language. */
  title: string;
  /** How many of its values are chosen. */
  chosen: number;
  options: Option[];
  /** The menu's search: the options whose name holds it, whatever the case. */
  search: string;
  /** Whether more than one option may be chosen; a filter of days takes one range. */
  multiple?: boolean;
};

/** A filter of the menu: its name, which hides or shows its options, and the options its search lets through. */
function Section({ title, chosen, options, search, multiple = true }: SectionProps) {
  const { t } = useTranslation();
  const [shown, setShown] = useState(true);
  const matching = options.filter((option) => option.label.toLowerCase().includes(search.toLowerCase()));
  return (
    <div className="py-2">
      <FilterHeader
        title={chosen > 0 ? `${title} (${chosen})` : title}
        isPreviewEnabled={shown}
        handleIsPreviewEnabled={() => setShown(!shown)}
      />
      {shown &&
        (matching.length > 0 ? (
          matching.map((option) => (
            <FilterOption
              key={option.key}
              isChecked={option.checked}
              onClick={option.pick}
              icon={option.icon}
              title={option.label}
              multiple={multiple}
            />
          ))
        ) : (
          <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
        ))}
    </div>
  );
}

/** A filter of days: when the work items were submitted (created) or last changed (updated). */
type DayField = "created" | "updated";

/**
 * The options of the filters of the list of the intake page's tab (P5a spec 2.2, 2.7, R9, R10, R23): the statuses
 * the tab lists (tabStatuses), the last one chosen staying (toggledStatuses); the priorities; the assignees and the
 * creators, the route project's members (getProjectMemberIds, guests too: they submit); its labels; and, for when the
 * work items were submitted and last changed, one range of days each: a preset, turned into days of the caller's
 * calendar as it is chosen (presetRange), or days the custom dates' dialog chooses (customRange). A chosen option
 * turns off as it is pressed again. Each change is the page's (setFilters), from the filters it holds then; a search
 * narrows the options by name.
 */
const IntakeFilterSelection = observer(function IntakeFilterSelection({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const { filters, setFilters } = useIntakePageState();
  const {
    getUserDetails,
    project: { getProjectMemberIds },
  } = useMember();
  const { getProjectLabels } = useLabel();
  const [search, setSearch] = useState("");
  // the filter of days whose custom dates' dialog is open
  const [custom, setCustom] = useState<DayField | undefined>(undefined);
  const now = new Date();

  const members = (getProjectMemberIds(projectId, true) ?? []).flatMap((id) => {
    const member = getUserDetails(id);
    return member ? [{ id, name: member.display_name }] : [];
  });
  const memberOptions = (field: "assignees" | "creators"): Option[] =>
    members.map(
      ({ id, name }): Option => ({
        key: id,
        label: name,
        icon: <ButtonAvatars userIds={id} />,
        checked: filters[field].includes(id),
        pick: () => setFilters((current) => ({ ...current, [field]: toggledOptions(current[field], [id]) })),
      })
    );
  const dayOptions = (field: DayField): Option[] => {
    const range: DayRange | undefined = filters[field];
    const preset = presetOf(range, now);
    // days of the custom dates' dialog: a range no preset names
    const customChosen = range !== undefined && preset === undefined;
    return [
      ...INTAKE_DATE_PRESETS.map(({ value, i18n_label }) => ({
        key: value,
        label: t(i18n_label),
        checked: preset === value,
        pick: () =>
          setFilters((current) => ({
            ...current,
            [field]: presetOf(current[field], new Date()) === value ? undefined : presetRange(value, new Date()),
          })),
      })),
      {
        key: "custom",
        label: t("common.filter_options.custom"),
        checked: customChosen,
        pick: () => (customChosen ? setFilters((current) => ({ ...current, [field]: undefined })) : setCustom(field)),
      },
    ];
  };
  const dayTitle = (field: DayField) =>
    field === "created" ? t("inbox_issue.filters.created_at") : t("inbox_issue.filters.updated_at");

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      {custom !== undefined && (
        <DateFilterModal
          isOpen
          title={dayTitle(custom)}
          handleClose={() => setCustom(undefined)}
          onSelect={(values) => setFilters((current) => ({ ...current, [custom]: customRange(values) }))}
        />
      )}
      <div className="bg-surface-1 p-2.5 pb-0">
        <div className="flex items-center gap-1.5 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-1.5 py-1 text-11">
          <SearchOutline className="text-placeholder" width={12} height={12} aria-hidden="true" />
          <input
            type="text"
            className="w-full bg-surface-2 outline-none placeholder:text-placeholder"
            placeholder={t("common.filter_options.search")}
            aria-label={t("common.filter_options.search")}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          {search !== "" && (
            <button
              type="button"
              className="grid place-items-center"
              aria-label={t("common.search.clear")}
              onClick={() => setSearch("")}
            >
              <CloseOutline className="text-tertiary" height={12} width={12} aria-hidden="true" />
            </button>
          )}
        </div>
      </div>
      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        <Section
          title={t("inbox_issue.filters.status")}
          chosen={filters.statuses.length}
          search={search}
          options={tabStatuses(filters.tab).map((status) => ({
            key: status,
            label: t(INTAKE_STATUS_LABELS[status]),
            checked: filters.statuses.includes(status),
            pick: () =>
              setFilters((current) => {
                const statuses = toggledStatuses(current.statuses, status);
                return statuses === undefined ? current : { ...current, statuses };
              }),
          }))}
        />
        <Section
          title={t("common.priority")}
          chosen={filters.priorities.length}
          search={search}
          options={ISSUE_PRIORITIES.map(
            ({ key }): Option => ({
              key,
              label: t(key),
              icon: <PriorityIcon priority={key} className="h-3.5 w-3.5" />,
              checked: filters.priorities.includes(key),
              pick: () =>
                setFilters((current) => ({ ...current, priorities: toggledOptions(current.priorities, [key]) })),
            })
          )}
        />
        <Section
          title={t("common.assignees")}
          chosen={filters.assignees.length}
          search={search}
          options={memberOptions("assignees")}
        />
        <Section
          title={t("common.created_by")}
          chosen={filters.creators.length}
          search={search}
          options={memberOptions("creators")}
        />
        <Section
          title={t("common.labels")}
          chosen={filters.labels.length}
          search={search}
          options={(getProjectLabels(projectId) ?? []).map(
            (label): Option => ({
              key: label.id,
              label: label.name,
              icon: <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: label.color }} />,
              checked: filters.labels.includes(label.id),
              pick: () => setFilters((current) => ({ ...current, labels: toggledOptions(current.labels, [label.id]) })),
            })
          )}
        />
        {(["created", "updated"] as const).map((field) => (
          <Section
            key={field}
            title={dayTitle(field)}
            chosen={filters[field] === undefined ? 0 : 1}
            search={search}
            options={dayOptions(field)}
            multiple={false}
          />
        ))}
      </div>
    </div>
  );
});

/**
 * The filters' menu of the list of the intake page's tab (P5a spec 2.7; M3 closeout's FiltersDropdown): its button,
 * named Filters, says whether the menu is open, Escape closing it and giving the focus back; it marks filters of the
 * user's own (customized). Its options are the tab's (IntakeFilterSelection), mounted only while it is open.
 */
export const IntakeFiltersMenu = observer(function IntakeFiltersMenu({ projectId }: { projectId: string }) {
  const { filters } = useIntakePageState();
  return (
    <FiltersDropdown
      icon={<FilterOutline className="h-3 w-3" />}
      miniIcon={<FilterOutline className="h-3.5 w-3.5" aria-hidden="true" />}
      placement="bottom-end"
      isFiltersApplied={customized(filters)}
    >
      <IntakeFilterSelection projectId={projectId} />
    </FiltersDropdown>
  );
});
