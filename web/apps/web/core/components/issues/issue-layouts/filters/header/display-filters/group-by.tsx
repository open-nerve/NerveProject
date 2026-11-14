/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@nerve/types";
// hooks
import { useRouteListWords } from "@/hooks/use-route-list-words";
// helpers
import { useGroupByOptions } from "../../../utils";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

type Props = {
  displayFilters: IIssueDisplayFilterOptions | undefined;
  groupByOptions: TIssueGroupByOptions[];
  handleUpdate: (val: TIssueGroupByOptions) => void;
};

/**
 * The groupings a list page offers (P7b spec 2.5, R8): those of its options; on a page of a project the cycle and the
 * module only while the route's project has their domain turned on, but the saved one always: nerve still groups by
 * it, and here it is changed. The panel never changes a saved grouping by itself.
 */
export const FilterGroupBy = observer(function FilterGroupBy(props: Props) {
  const { displayFilters, groupByOptions, handleUpdate } = props;
  // hooks
  const { t } = useTranslation();
  const offered = useRouteListWords();
  const [previewEnabled, setPreviewEnabled] = useState(true);

  const selectedGroupBy = displayFilters?.group_by ?? null;
  const selectedSubGroupBy = displayFilters?.sub_group_by ?? null;

  const options = useGroupByOptions(groupByOptions).filter(({ key }) => key === selectedGroupBy || offered(key));

  return (
    <>
      <FilterHeader
        title={t("common.group_by")}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {options.map((groupBy) => {
            if (
              displayFilters?.layout === "kanban" &&
              selectedSubGroupBy !== null &&
              groupBy.key === selectedSubGroupBy
            )
              return null;

            return (
              <FilterOption
                key={groupBy?.key}
                isChecked={selectedGroupBy === groupBy?.key}
                onClick={() => handleUpdate(groupBy.key)}
                title={t(groupBy.titleTranslationKey)}
                multiple={false}
              />
            );
          })}
        </div>
      )}
    </>
  );
});
