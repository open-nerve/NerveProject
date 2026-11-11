/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import { useState } from "react";
import { observer } from "mobx-react";
import { CircleDashed } from "lucide-react";
import { AddOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
// types
import type { TIssueGroupByOptions } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
// ui
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// constants
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";

interface IHeaderGroupByCard {
  groupID: string;
  groupBy: TIssueGroupByOptions;
  icon?: React.ReactNode;
  title: string;
  count: number;
  issuePayload: Partial<IssueDetail>;
  disableIssueCreation?: boolean;
  isExpanded: boolean;
  handleCollapsedGroups: (value: string) => void;
}

export const HeaderGroupByCard = observer(function HeaderGroupByCard(props: IHeaderGroupByCard) {
  const {
    groupID,
    groupBy,
    icon,
    title,
    count,
    issuePayload,
    disableIssueCreation,
    isExpanded,
    handleCollapsedGroups,
  } = props;
  // states
  const [isOpen, setIsOpen] = useState(false);
  const storeType = useIssueStoreType();
  const { t } = useTranslation();
  const heading = (
    <>
      <span className="line-clamp-1 inline-block truncate font-medium text-primary">{title}</span>
      <span className="pl-2 text-13 font-medium text-tertiary">{count || 0}</span>
      <span className="px-2.5" />
    </>
  );
  return (
    <>
      <div className="group/list-header flex w-full flex-shrink-0 items-center gap-2 py-1.5">
        <div className="grid flex-shrink-0 place-items-center overflow-hidden">
          {icon ?? <CircleDashed className="size-3.5" strokeWidth={2} />}
        </div>

        {groupBy ? (
          <button
            type="button"
            aria-label={title}
            aria-expanded={isExpanded}
            className="relative flex w-full cursor-pointer flex-row items-center gap-1 overflow-hidden rounded-xs text-left"
            onClick={() => handleCollapsedGroups(groupID)}
          >
            {heading}
          </button>
        ) : (
          <div className="relative flex w-full flex-row items-center gap-1 overflow-hidden">{heading}</div>
        )}

        {!disableIssueCreation && (
          <button
            type="button"
            aria-label={`${t("create_work_item")}: ${title}`}
            className="flex h-5 w-5 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-xs transition-all hover:bg-layer-1"
            onClick={() => {
              setIsOpen(true);
            }}
          >
            <AddOutline width={14} />
          </button>
        )}

        <CreateUpdateIssueModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          data={issuePayload}
          storeType={storeType}
          // a group's new work item of a cycle's page joins the cycle, so it stays in the cycle's project (P5b spec 2.5)
          isProjectSelectionDisabled={storeType === EIssuesStoreType.CYCLE}
        />
      </div>
    </>
  );
});
