/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import type React from "react";
import { observer } from "mobx-react";
// lucide icons
import { Circle } from "lucide-react";
import { AddOutline, ArrowCollapseOutline, FullScreenOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import type { GroupJoinRefusal, TIssueKanbanFilters, TIssueGroupByOptions } from "@nerve/types";
// components
import { isDialogStore, useCreateDialogRequest } from "@/components/issues/issue-layouts/command-host";
import { GroupAddExisting } from "@/components/issues/issue-layouts/group-add-existing";
import { useGroupName } from "@/components/issues/issue-layouts/group-columns";
// hooks
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";

interface IHeaderGroupByCard {
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  column_id: string;
  icon?: React.ReactNode;
  title: string;
  /** An archived cycle's or module's group, named so (P7b R4). */
  archived?: boolean;
  /** Why nerve refuses a work item's joining the column, which then offers no new work item (P7b R4). */
  refusal?: GroupJoinRefusal;
  count: number;
  collapsedGroups: TIssueKanbanFilters;
  handleCollapsedGroups: (toggle: "group_by" | "sub_group_by", value: string) => void;
  issuePayload: Partial<IssueDetail>;
  disableIssueCreation?: boolean;
}

export const HeaderGroupByCard = observer(function HeaderGroupByCard(props: IHeaderGroupByCard) {
  const {
    sub_group_by,
    column_id,
    icon,
    title,
    archived,
    refusal,
    count,
    collapsedGroups,
    handleCollapsedGroups,
    issuePayload,
    disableIssueCreation,
  } = props;
  const verticalAlignPosition = sub_group_by ? false : collapsedGroups?.group_by.includes(column_id);
  // hooks
  const storeType = useIssueStoreType();
  const requestCreate = useCreateDialogRequest();
  const { t } = useTranslation();
  const { label, caption } = useGroupName(title, archived);
  // what the header adds work items by, its "+" and beside it "add existing" (P7b spec 2.9, R20), on one gate
  const adding = !disableIssueCreation && !refusal;
  return (
    <div
      className={`relative flex flex-shrink-0 gap-1 py-1.5 ${
        verticalAlignPosition ? `w-[44px] flex-col items-center` : `w-full flex-row items-center`
      }`}
    >
      <div className="flex size-5 flex-shrink-0 items-center justify-center overflow-hidden rounded-xs">
        {icon ? icon : <Circle width={14} strokeWidth={2} />}
      </div>

      <div
        className={`relative flex gap-1 ${
          verticalAlignPosition ? `flex-col items-center` : `w-full flex-row items-baseline overflow-hidden`
        }`}
      >
        <div
          className={`line-clamp-1 inline-block truncate overflow-hidden font-medium text-primary ${
            verticalAlignPosition ? `max-h-[400px] vertical-lr` : ``
          }`}
        >
          {title}
          {caption}
        </div>
        <div className={`flex-shrink-0 text-13 font-medium text-tertiary ${verticalAlignPosition ? `pr-0.5` : `pl-2`}`}>
          {count || 0}
        </div>
      </div>

      {sub_group_by === null && (
        <button
          type="button"
          aria-label={label}
          aria-expanded={!verticalAlignPosition}
          className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover"
          onClick={() => handleCollapsedGroups("group_by", column_id)}
        >
          {verticalAlignPosition ? <FullScreenOutline width={14} /> : <ArrowCollapseOutline width={14} />}
        </button>
      )}

      {adding && (
        <button
          type="button"
          aria-label={`${t("create_work_item")}: ${label}`}
          className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover"
          onClick={() => {
            // the layout's command host holds the dialog, which outlives the column while the list is checked
            // (P7a spec 2.7, R16)
            if (!requestCreate || !isDialogStore(storeType))
              throw new Error("A group's new work item requires an IssueListCommandHost");
            requestCreate(issuePayload, storeType);
          }}
        >
          <AddOutline width={14} />
        </button>
      )}

      {adding && (
        <GroupAddExisting
          label={label}
          className="flex h-[20px] w-[20px] flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover"
        />
      )}
    </div>
  );
});
