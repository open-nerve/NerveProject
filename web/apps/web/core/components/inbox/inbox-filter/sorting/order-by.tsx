/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ChevronDownOutline, SortAscendingOutline, SortDescendingOutline, TickOutline } from "@makeplane/propel/icons";
import { INBOX_ISSUE_ORDER_BY_OPTIONS, INBOX_ISSUE_SORT_BY_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import type { TInboxIssueSortingOrderByKeys, TInboxIssueSortingSortByKeys } from "@nerve/types";
import { CustomMenu } from "@nerve/ui";
// helpers
import { cn } from "@nerve/utils";
// hooks
import { useProjectInbox } from "@/hooks/store/use-project-inbox";
import useSize from "@/hooks/use-window-size";

export const InboxIssueOrderByDropdown = observer(function InboxIssueOrderByDropdown() {
  // hooks
  const { t } = useTranslation();
  const windowSize = useSize();
  const { inboxSorting, handleInboxIssueSorting } = useProjectInbox();
  const orderByDetails =
    INBOX_ISSUE_ORDER_BY_OPTIONS.find((option) => inboxSorting?.order_by?.includes(option.key)) || undefined;
  const smallButton =
    inboxSorting?.sort_by === "asc" ? (
      <SortAscendingOutline className="size-3" />
    ) : (
      <SortDescendingOutline className="size-3" />
    );
  const largeButton = (
    <div className={cn(getButtonStyling("secondary", "base"), "px-2 text-tertiary")}>
      {inboxSorting?.sort_by === "asc" ? (
        <SortAscendingOutline className="size-3" />
      ) : (
        <SortDescendingOutline className="size-3" />
      )}
      {t(orderByDetails?.i18n_label || "inbox_issue.order_by.created_at")}
      <ChevronDownOutline className="size-3" />
    </div>
  );
  return (
    <CustomMenu
      customButton={windowSize[0] > 1280 ? largeButton : smallButton}
      placement="bottom-end"
      maxHeight="lg"
      closeOnSelect
    >
      {INBOX_ISSUE_ORDER_BY_OPTIONS.map((option) => (
        <CustomMenu.MenuItem
          key={option.key}
          className="flex items-center justify-between gap-2"
          onClick={() => handleInboxIssueSorting("order_by", option.key as TInboxIssueSortingOrderByKeys)}
        >
          {t(option.i18n_label)}
          {inboxSorting?.order_by?.includes(option.key) && <TickOutline className="size-3" />}
        </CustomMenu.MenuItem>
      ))}
      <hr className="my-2 border-subtle" />
      {INBOX_ISSUE_SORT_BY_OPTIONS.map((option) => (
        <CustomMenu.MenuItem
          key={option.key}
          className="flex items-center justify-between gap-2"
          onClick={() => handleInboxIssueSorting("sort_by", option.key as TInboxIssueSortingSortByKeys)}
        >
          {t(option.i18n_label)}
          {inboxSorting?.sort_by?.includes(option.key) && <TickOutline className="size-3" />}
        </CustomMenu.MenuItem>
      ))}
    </CustomMenu>
  );
});
