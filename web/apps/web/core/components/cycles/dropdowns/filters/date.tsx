/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { DATE_AFTER_FILTER_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { isInDateFormat } from "@nerve/utils";
// components
import { DateFilterModal } from "@/components/core/filters/date-filter-modal";
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

type Props = {
  /** What the days filtered are, in the page's language: the day a cycle starts, or ends. */
  title: string;
  appliedFilters: readonly string[] | undefined;
  /** Turns each of the options on, or off if on. */
  handleUpdate: (options: readonly string[]) => void;
  searchQuery: string;
};

/**
 * The options of the filters' menu for a day of a cycle (P5a spec 2.3): some weeks or months from today at the
 * earliest, and a custom day, which a dialog picks (and the option, on, turns off).
 */
export const FilterDate = observer(function FilterDate(props: Props) {
  const { title, appliedFilters, handleUpdate, searchQuery } = props;
  // hooks
  const { t } = useTranslation();
  // states
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const [isDateFilterModalOpen, setIsDateFilterModalOpen] = useState(false);

  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const filteredOptions = DATE_AFTER_FILTER_OPTIONS.filter((d) =>
    d.name.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const customDays = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0] ?? "")) ?? [];
  const handleCustomDate = () => {
    if (customDays.length > 0) handleUpdate(customDays);
    else setIsDateFilterModalOpen(true);
  };

  return (
    <>
      {isDateFilterModalOpen && (
        <DateFilterModal
          handleClose={() => setIsDateFilterModalOpen(false)}
          isOpen={isDateFilterModalOpen}
          onSelect={(val) => handleUpdate(val)}
          title={title}
        />
      )}
      <FilterHeader
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {filteredOptions.length > 0 ? (
            <>
              {filteredOptions.map((option) => (
                <FilterOption
                  key={option.value}
                  isChecked={!!appliedFilters?.includes(option.value)}
                  onClick={() => handleUpdate([option.value])}
                  title={option.name}
                  multiple
                />
              ))}
              <FilterOption
                isChecked={customDays.length > 0}
                onClick={handleCustomDate}
                title={t("project_cycles.filters.custom")}
                multiple
              />
            </>
          ) : (
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
          )}
        </div>
      )}
    </>
  );
});
