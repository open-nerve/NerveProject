/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { observer } from "mobx-react";
import { ISSUE_GROUP_BY_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@nerve/types";
// hooks
import { useRouteListWords } from "@/hooks/use-route-list-words";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

type Props = {
  displayFilters: IIssueDisplayFilterOptions;
  handleUpdate: (val: TIssueGroupByOptions) => void;
  subGroupByOptions: TIssueGroupByOptions[];
};

/**
 * The sub-groupings a board offers (P7b spec 2.5, R8): those of its options but its grouping; on a page of a project the
 * cycle and the module only while the route's project has their domain turned on, but the saved one always, as the
 * grouping panel (group-by.tsx).
 */
export const FilterSubGroupBy = observer(function FilterSubGroupBy(props: Props) {
  // hooks
  const { t } = useTranslation();
  const offered = useRouteListWords();

  const { displayFilters, handleUpdate, subGroupByOptions } = props;

  const [previewEnabled, setPreviewEnabled] = useState(true);

  const selectedGroupBy = displayFilters.group_by ?? null;
  const selectedSubGroupBy = displayFilters.sub_group_by ?? null;

  return (
    <>
      <FilterHeader
        title="Sub-group by"
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {ISSUE_GROUP_BY_OPTIONS.filter(
            ({ key }) => subGroupByOptions.includes(key) && (key === selectedSubGroupBy || offered(key))
          ).map((subGroupBy) => {
            if (selectedGroupBy !== null && subGroupBy.key === selectedGroupBy) return null;

            return (
              <FilterOption
                key={subGroupBy?.key}
                isChecked={selectedSubGroupBy === subGroupBy?.key}
                onClick={() => handleUpdate(subGroupBy.key)}
                title={t(subGroupBy.titleTranslationKey)}
                multiple={false}
              />
            );
          })}
        </div>
      )}
    </>
  );
});
