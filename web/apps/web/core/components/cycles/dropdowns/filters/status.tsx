/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { CycleStatus } from "@nerve/api-client";
import { CYCLE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

type Props = {
  appliedFilters: readonly CycleStatus[] | undefined;
  /** Turns the status on, or off if on. */
  handleUpdate: (status: CycleStatus) => void;
  searchQuery: string;
};

/** The options of the filters' menu for nerve's status of a cycle (P5a spec 2.3), each by its name in the page. */
export const FilterStatus = observer(function FilterStatus(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery } = props;
  // states
  const [previewEnabled, setPreviewEnabled] = useState(true);
  // hooks
  const { t } = useTranslation();
  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const title = t("project_cycles.filters.status");
  const filteredOptions = CYCLE_STATUS.filter((status) =>
    t(status.i18n_title).toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <>
      <FilterHeader
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {filteredOptions.length > 0 ? (
            filteredOptions.map((status) => (
              <FilterOption
                key={status.value}
                isChecked={!!appliedFilters?.includes(status.value)}
                onClick={() => handleUpdate(status.value)}
                title={t(status.i18n_title)}
              />
            ))
          ) : (
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
          )}
        </div>
      )}
    </>
  );
});
