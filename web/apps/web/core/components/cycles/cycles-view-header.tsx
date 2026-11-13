/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { FilterOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { calculateTotalFilters } from "@nerve/utils";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { PlanningListSearch } from "@/components/planning/list-search";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
// store
import type { CycleList } from "@/store/planning/cycle_filter.store";
// local imports
import { CycleFiltersSelection } from "./dropdowns";

type Props = {
  projectId: string;
  /** The project's list the search and the filters are of: its cycles list, or its archive. */
  list: CycleList;
};

/**
 * The search and the filters' menu of a project's cycles list or archive (P5a spec 2.3, 2.5), kept by the project and
 * the list in the session's filters. The search is the one the modules list has too (PlanningListSearch).
 */
export const CyclesViewHeader = observer(function CyclesViewHeader(props: Props) {
  const { projectId, list } = props;
  // hooks
  const { t } = useTranslation();
  const { getFilters, getSearchQuery, updateSearchQuery } = useCycleFilters();

  return (
    <div className="flex items-center gap-2">
      <PlanningListSearch
        query={getSearchQuery(projectId, list)}
        onSearch={(query) => updateSearchQuery(projectId, list, query)}
      />
      <FiltersDropdown
        icon={<FilterOutline className="h-3 w-3" />}
        title={t("common.filters")}
        placement="bottom-end"
        isFiltersApplied={calculateTotalFilters(getFilters(projectId, list)) !== 0}
      >
        <CycleFiltersSelection projectId={projectId} list={list} />
      </FiltersDropdown>
    </div>
  );
});
