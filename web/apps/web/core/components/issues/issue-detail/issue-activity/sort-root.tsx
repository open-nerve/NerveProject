/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { memo } from "react";
import { SortAscendingOutline, SortDescendingOutline } from "@makeplane/propel/icons";
// nerve package imports
import { E_SORT_ORDER } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";

export type TActivitySortRoot = {
  sortOrder: E_SORT_ORDER;
  toggleSort: () => void;
};
export const ActivitySortRoot = memo(function ActivitySortRoot(props: TActivitySortRoot) {
  const { t } = useTranslation();
  const SortIcon = props.sortOrder === E_SORT_ORDER.ASC ? SortAscendingOutline : SortDescendingOutline;
  return (
    <IconButton
      variant="tertiary"
      icon={SortIcon}
      aria-label={t("aria_labels.toggle_sort_order")}
      onClick={props.toggleSort}
    />
  );
});

ActivitySortRoot.displayName = "ActivitySortRoot";
