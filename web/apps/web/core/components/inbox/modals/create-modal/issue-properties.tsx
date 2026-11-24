/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedPayloadDate, getDate } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { IssueLabelSelect } from "@/components/issues/select";

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
  const { t } = useTranslation();

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
          placeholder={t("issue.add.assignee")}
          multiple
        />
      </div>

      {/* labels */}
      <div className="h-7">
        <IssueLabelSelect
          issueName={data.name}
          value={data?.label_ids || []}
          onChange={(labelIds) => handleData("label_ids", labelIds)}
          projectId={projectId}
        />
      </div>

      {/* due date */}
      <div className="h-7">
        <DateDropdown
          value={data?.target_date || null}
          onChange={(date) => handleData("target_date", date ? renderFormattedPayloadDate(date) : "")}
          buttonVariant="border-with-text"
          minDate={minDate ?? undefined}
          placeholder={t("issue.add.due_date")}
        />
      </div>
    </div>
  );
});
