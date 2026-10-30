/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useLayoutEffect, useMemo, useState } from "react";
import { observer } from "mobx-react";
import { v4 as uuidv4 } from "uuid";
// nerve imports
import type { TSaveViewOptions, TUpdateViewOptions } from "@nerve/constants";
import type { IWorkItemFilterInstance } from "@nerve/shared-state";
import type { IIssueFilters, TWorkItemFilterExpression } from "@nerve/types";
// store hooks
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
// hooks
import type { TWorkItemFiltersEntityProps } from "@/hooks/work-item-filters/use-work-item-filters-config";
import { useWorkItemFiltersConfig } from "@/hooks/work-item-filters/use-work-item-filters-config";
// local imports
import type { TSharedWorkItemFiltersHOCProps, TSharedWorkItemFiltersProps } from "./shared";

type TAdditionalWorkItemFiltersProps = {
  saveViewOptions?: TSaveViewOptions<TWorkItemFilterExpression>;
  updateViewOptions?: TUpdateViewOptions<TWorkItemFilterExpression>;
} & TWorkItemFiltersEntityProps;

type TWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & TAdditionalWorkItemFiltersProps;

export const WorkItemFiltersHOC = observer(function WorkItemFiltersHOC(props: TWorkItemFiltersHOCProps) {
  const { children, initialWorkItemFilters } = props;

  // Only initialize filter instance when initial work item filters are defined
  if (!initialWorkItemFilters)
    return <>{typeof children === "function" ? children({ filter: undefined }) : children}</>;

  return (
    <WorkItemFilterRoot {...props} initialWorkItemFilters={initialWorkItemFilters}>
      {children}
    </WorkItemFilterRoot>
  );
});

type TWorkItemFilterProps = TSharedWorkItemFiltersProps &
  TAdditionalWorkItemFiltersProps & {
    initialWorkItemFilters: IIssueFilters;
    children: React.ReactNode | ((props: { filter: IWorkItemFilterInstance }) => React.ReactNode);
  };

const WorkItemFilterRoot = observer(function WorkItemFilterRoot(props: TWorkItemFilterProps) {
  const {
    children,
    entityType,
    entityId,
    filtersToShowByLayout,
    initialWorkItemFilters,
    isTemporary,
    saveViewOptions,
    updateFilters,
    updateViewOptions,
    showOnMount,
    ...entityConfigProps
  } = props;
  // store hooks
  const store = useWorkItemFilters();
  const { getFilter, getOrCreateFilter, deleteFilter } = store;
  // derived values
  const workItemEntityID = useMemo(
    () => (isTemporary ? `TEMP-${entityId ?? uuidv4()}` : entityId),
    [isTemporary, entityId]
  );
  // memoize initial values to prevent re-computations when reference changes
  const initialUserFilters = useMemo(() => initialWorkItemFilters.richFilters, [initialWorkItemFilters]);
  const workItemFiltersConfig = useWorkItemFiltersConfig({
    allowedFilters: filtersToShowByLayout ? filtersToShowByLayout : [],
    ...entityConfigProps,
  });
  const key = JSON.stringify([entityType, workItemEntityID]);
  const [current, setCurrent] = useState<{ store: typeof store; key: string; filter: IWorkItemFilterInstance }>();
  // Registry writes and option notifications belong to the commit phase, never a render memo.
  useLayoutEffect(() => {
    const filter = getOrCreateFilter({
      entityType,
      entityId: workItemEntityID,
      initialExpression: initialUserFilters,
      onExpressionChange: updateFilters,
      expressionOptions: { saveViewOptions, updateViewOptions },
      showOnMount,
    });
    setCurrent({ store, key, filter });
  }, [
    getOrCreateFilter,
    store,
    entityType,
    workItemEntityID,
    initialUserFilters,
    updateFilters,
    saveViewOptions,
    updateViewOptions,
    showOnMount,
    key,
  ]);
  useLayoutEffect(() => {
    const owned = getFilter(entityType, workItemEntityID);
    return () => {
      if (getFilter(entityType, workItemEntityID) === owned) deleteFilter(entityType, workItemEntityID);
    };
  }, [getFilter, deleteFilter, entityType, workItemEntityID]);
  const workItemLayoutFilter = current?.store === store && current.key === key ? current.filter : undefined;

  useEffect(() => {
    if (!workItemLayoutFilter) return;
    workItemLayoutFilter.configManager.setAreConfigsReady(workItemFiltersConfig.areAllConfigsInitialized);
    workItemLayoutFilter.configManager.registerAll(workItemFiltersConfig.configs);
  }, [workItemFiltersConfig.areAllConfigsInitialized, workItemFiltersConfig.configs, workItemLayoutFilter]);

  if (!workItemLayoutFilter) return null;
  return <>{typeof children === "function" ? children({ filter: workItemLayoutFilter }) : children}</>;
});
