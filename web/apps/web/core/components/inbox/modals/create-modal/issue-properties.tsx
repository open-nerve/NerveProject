/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import { observer } from "mobx-react";
import { ETabIndices } from "@nerve/constants";
import { renderFormattedPayloadDate, getDate, getTabIndex } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { IssueLabelSelect } from "@/components/issues/select";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";

type TInboxIssueProperties = {
  projectId: string;
  data: Partial<IssueDetail>;
  handleData: (
    issueKey: keyof Partial<IssueDetail>,
    issueValue: Partial<IssueDetail>[keyof Partial<IssueDetail>]
  ) => void;
};

export const InboxIssueProperties = observer(function InboxIssueProperties(props: TInboxIssueProperties) {
  const { projectId, data, handleData } = props;
  // hooks
  const { isMobile } = usePlatformOS();
  const { getIndex } = getTabIndex(ETabIndices.INTAKE_ISSUE_FORM, isMobile);

  const startDate = data?.start_date;

  const minDate = getDate(startDate);
  minDate?.setDate(minDate.getDate());

  return (
    <div className="relative flex flex-wrap items-center gap-2">
      {/* priority */}
      <div className="h-7">
        <PriorityDropdown
          value={data?.priority}
          onChange={(priority) => handleData("priority", priority)}
          buttonVariant="border-with-text"
          tabIndex={getIndex("priority")}
        />
      </div>

      {/* Assignees */}
      <div className="h-7">
        <MemberDropdown
          projectId={projectId}
          value={data?.assignee_ids || []}
          onChange={(assigneeIds) => handleData("assignee_ids", assigneeIds)}
          buttonVariant={(data?.assignee_ids || [])?.length > 0 ? "transparent-without-text" : "border-with-text"}
          buttonClassName={(data?.assignee_ids || [])?.length > 0 ? "hover:bg-transparent" : ""}
          placeholder="Assignees"
          multiple
          tabIndex={getIndex("assignee_ids")}
        />
      </div>

      {/* labels */}
      <div className="h-7">
        <IssueLabelSelect
          issueName={data.name}
          value={data?.label_ids || []}
          onChange={(labelIds) => handleData("label_ids", labelIds)}
          projectId={projectId}
          tabIndex={getIndex("label_ids")}
        />
      </div>

      {/* due date */}
      <div className="h-7">
        <DateDropdown
          value={data?.target_date || null}
          onChange={(date) => handleData("target_date", date ? renderFormattedPayloadDate(date) : "")}
          buttonVariant="border-with-text"
          minDate={minDate ?? undefined}
          placeholder="Due date"
          tabIndex={getIndex("target_date")}
        />
      </div>
    </div>
  );
});
