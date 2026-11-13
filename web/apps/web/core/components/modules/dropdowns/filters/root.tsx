/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { FilterDate } from "@/components/planning/filter-date";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useModuleFilters } from "@/hooks/store/use-module-filters";
// store
import type { ModuleListKind } from "@/store/planning/module_filter.store";
// local imports
import { FilterMembers } from "./members";
import { FilterStatus } from "./status";

/** The options held, with each of options turned on, or off if it was on. */
const toggled = <T extends string>(held: readonly T[] | undefined, options: readonly T[]): T[] => {
  const next = new Set<T>(held);
  for (const option of options) {
    if (next.has(option)) next.delete(option);
    else next.add(option);
  }
  return [...next];
};

type Props = {
  /** The workspace of the page's address. */
  workspaceSlug: string;
  projectId: string;
  /** The project's list the filters are of: its modules list, or its archive. */
  list: ModuleListKind;
};

/**
 * The filters' menu of a project's modules list or archive (P6a spec 2.4, R14): nerve's status (the list's alone, as
 * the old page had it), the lead and the members, who are the members of the workspace of the page's address
 * (getWorkspaceMemberIds, by its slug: never the router's), the day a module starts and the day it is due; a search of
 * its options. Each option picked turns on, or off if on, in the project's filters of the list (the session's). The
 * menu opens on its button, from which Tab goes into it (FiltersDropdown).
 */
export const ModuleFiltersSelection = observer(function ModuleFiltersSelection(props: Props) {
  const { workspaceSlug, projectId, list } = props;
  // states
  const [filtersSearchQuery, setFiltersSearchQuery] = useState("");
  // hooks
  const { t } = useTranslation();
  const { getFilters, updateFilters } = useModuleFilters();
  const {
    workspace: { getWorkspaceMemberIds },
  } = useMember();
  const filters = getFilters(projectId, list);
  const memberIds = getWorkspaceMemberIds(workspaceSlug);

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="bg-surface-1 p-2.5 pb-0">
        <div className="flex items-center gap-1.5 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-1.5 py-1 text-11">
          <SearchOutline className="text-placeholder" width={12} height={12} aria-hidden="true" />
          <input
            type="text"
            className="w-full bg-surface-2 outline-none placeholder:text-placeholder"
            placeholder={t("common.filter_options.search")}
            aria-label={t("common.filter_options.search")}
            value={filtersSearchQuery}
            onChange={(e) => setFiltersSearchQuery(e.target.value)}
          />
          {filtersSearchQuery !== "" && (
            <button
              type="button"
              className="grid place-items-center"
              aria-label={t("common.search.clear")}
              onClick={() => setFiltersSearchQuery("")}
            >
              <CloseOutline className="text-tertiary" height={12} width={12} aria-hidden="true" />
            </button>
          )}
        </div>
      </div>
      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        {list === "default" && (
          <div className="py-2">
            <FilterStatus
              appliedFilters={filters.status}
              handleUpdate={(status) => updateFilters(projectId, list, { status: toggled(filters.status, [status]) })}
              searchQuery={filtersSearchQuery}
            />
          </div>
        )}
        <div className="py-2">
          <FilterMembers
            title={t("lead")}
            appliedFilters={filters.lead}
            handleUpdate={(lead) => updateFilters(projectId, list, { lead: toggled(filters.lead, [lead]) })}
            memberIds={memberIds}
            searchQuery={filtersSearchQuery}
          />
        </div>
        <div className="py-2">
          <FilterMembers
            title={t("members")}
            appliedFilters={filters.members}
            handleUpdate={(member) => updateFilters(projectId, list, { members: toggled(filters.members, [member]) })}
            memberIds={memberIds}
            searchQuery={filtersSearchQuery}
          />
        </div>
        <div className="py-2">
          <FilterDate
            title={t("start_date")}
            appliedFilters={filters.start_date}
            handleUpdate={(days) => updateFilters(projectId, list, { start_date: toggled(filters.start_date, days) })}
            searchQuery={filtersSearchQuery}
          />
        </div>
        <div className="py-2">
          <FilterDate
            title={t("due_date")}
            appliedFilters={filters.target_date}
            handleUpdate={(days) => updateFilters(projectId, list, { target_date: toggled(filters.target_date, days) })}
            searchQuery={filtersSearchQuery}
          />
        </div>
      </div>
    </div>
  );
});
