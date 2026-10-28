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
// types
import type { TIssueGroupByOptions } from "@nerve/types";
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
  handleCollapsedGroups: (value: string) => void;
}

export const HeaderGroupByCard = observer(function HeaderGroupByCard(props: IHeaderGroupByCard) {
  const { groupID, icon, title, count, issuePayload, disableIssueCreation, handleCollapsedGroups } = props;
  // states
  const [isOpen, setIsOpen] = useState(false);
  const storeType = useIssueStoreType();
  return (
    <>
      <div className="group/list-header flex w-full flex-shrink-0 items-center gap-2 py-1.5">
        <div className="grid flex-shrink-0 place-items-center overflow-hidden">
          {icon ?? <CircleDashed className="size-3.5" strokeWidth={2} />}
        </div>

        {/* eslint-disable-next-line jsx_a11y/click-events-have-key-events eslint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div
          className="relative flex w-full cursor-pointer flex-row items-center gap-1 overflow-hidden"
          onClick={() => handleCollapsedGroups(groupID)}
        >
          <div className="line-clamp-1 inline-block truncate font-medium text-primary">{title}</div>
          <div className="pl-2 text-13 font-medium text-tertiary">{count || 0}</div>
          <div className="px-2.5"></div>
        </div>

        {!disableIssueCreation && (
          // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
          <div
            className="flex h-5 w-5 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-xs transition-all hover:bg-layer-1"
            onClick={() => {
              setIsOpen(true);
            }}
          >
            <AddOutline width={14} />
          </div>
        )}

        <CreateUpdateIssueModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          data={issuePayload}
          storeType={storeType}
        />
      </div>
    </>
  );
});
