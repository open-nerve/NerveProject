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
import { FilterCreatedDate } from "@/components/common/filters/created-at";
import { FilterCreatedBy } from "@/components/common/filters/created-by";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import type { ViewListFilters } from "../view-list-order";

type Props = {
  /** The filters applied to the list. */
  filters: ViewListFilters;
  /** Applies these filters instead. */
  onChange: (filters: ViewListFilters) => void;
  /** Those whose views the list may be filtered by: the project's members. */
  memberIds: string[] | undefined;
};

/**
 * The panel of a project's views list's filters (P6a spec 2.3): the dates of their creation and their owners, which a
 * search narrows; each option chosen is applied, chosen again, it is not.
 */
export const ViewFiltersSelection = observer(function ViewFiltersSelection(props: Props) {
  const { filters, onChange, memberIds } = props;
  // nerve hooks
  const { t } = useTranslation();
  // states
  const [filtersSearchQuery, setFiltersSearchQuery] = useState("");
  // store
  const { isMobile } = usePlatformOS();

  /** Applies the filter of key with value, or each of values, if it did not, and without it if it did. */
  const toggle = (key: keyof ViewListFilters, value: string | string[]) => {
    const values = Array.isArray(value) ? value : [value];
    const kept = filters[key].filter((each) => !values.includes(each));
    const added = values.filter((each) => !filters[key].includes(each));
    onChange({ ...filters, [key]: [...kept, ...added] });
  };

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="bg-surface-1 p-2.5 pb-0">
        <div className="flex items-center gap-1.5 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-1.5 py-1 text-11">
          <SearchOutline className="text-placeholder" width={12} height={12} aria-hidden="true" />
          <input
            type="text"
            className="w-full bg-surface-2 outline-none placeholder:text-placeholder"
            placeholder={t("common.search.label")}
            aria-label={t("common.search.label")}
            value={filtersSearchQuery}
            onChange={(e) => setFiltersSearchQuery(e.target.value)}
            // oxlint-disable-next-line jsx_a11y/no-autofocus
            autoFocus={!isMobile}
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
        {/* created date */}
        <div className="py-2">
          <FilterCreatedDate
            appliedFilters={filters.created_at}
            handleUpdate={(value) => toggle("created_at", value)}
            searchQuery={filtersSearchQuery}
          />
        </div>

        {/* created by */}
        <div className="py-2">
          <FilterCreatedBy
            appliedFilters={filters.owned_by}
            handleUpdate={(value) => toggle("owned_by", value)}
            searchQuery={filtersSearchQuery}
            memberIds={memberIds}
          />
        </div>
      </div>
    </div>
  );
});
