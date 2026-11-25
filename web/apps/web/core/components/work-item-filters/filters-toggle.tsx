/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { EIssuesStoreType } from "@nerve/types";
// components
import { FiltersToggle } from "@/components/rich-filters/filters-toggle";
// hooks
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";

type TWorkItemFiltersToggleProps = {
  entityType: EIssuesStoreType;
  entityId: string;
  /** The page's draft is being saved (FiltersToggle's). */
  disabled?: boolean;
};

export const WorkItemFiltersToggle = observer(function WorkItemFiltersToggle(props: TWorkItemFiltersToggleProps) {
  const { entityType, entityId, disabled } = props;
  // store hooks
  const { getFilter } = useWorkItemFilters();
  // derived values
  const filter = getFilter(entityType, entityId);

  return <FiltersToggle filter={filter} disabled={disabled} />;
});
