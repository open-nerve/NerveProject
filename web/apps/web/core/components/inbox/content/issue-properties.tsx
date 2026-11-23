/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { DueDateOutline, LabelsOutline, MembersOutline, PriorityOutline, StateOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import type { IssueChange } from "@nerve/types";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueLabel } from "@/components/issues/issue-detail/label";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// local imports
import { IntakeDuplicateOf } from "./duplicate-of";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  isEditable: boolean;
};

/**
 * The properties of the selected work item of a project's intake, as held (P5a spec 2.9): its assignees, priority, due
 * date and labels, each changed through the work item's queue of writes; its state only once it is accepted, as a work
 * item in the intake is in a state no catalog lists; and the work item it duplicates, if any.
 */
export const InboxIssueContentProperties = observer(function InboxIssueContentProperties(props: Props) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable } = props;
  const { t } = useTranslation();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const issue = getIssueDetailById(issueId);
  if (!issue) return null;
  // A property's change has no follow-up; the operation says its refusal.
  const update = (change: IssueChange) =>
    issueOperations.update(workspaceSlug, projectId, issueId, change).catch(() => undefined);
  const minDate = issue.start_date ? getDate(issue.start_date) : undefined;

  return (
    <div className="flex w-full flex-col divide-y-2 divide-subtle-1">
      <div className="w-full overflow-y-auto">
        <h5 className="mb-2 text-body-sm-medium">{t("common.properties")}</h5>
        <div className={`flex flex-col gap-3 ${!isEditable ? "opacity-60" : ""}`}>
          {issue.intake?.status === "accepted" && (
            <div className="flex h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <StateOutline className="h-4 w-4 flex-shrink-0" />
                <span>{t("common.state")}</span>
              </div>
              <StateDropdown
                value={issue.state_id}
                onChange={() => {}}
                projectId={projectId}
                disabled
                buttonVariant="transparent-with-text"
                className="group w-3/5 flex-grow"
                buttonContainerClassName="w-full text-left"
                buttonClassName="text-13"
                dropdownArrow
                dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
              />
            </div>
          )}
          <div className="flex h-8 items-center gap-2">
            <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
              <MembersOutline className="h-4 w-4 flex-shrink-0" />
              <span>{t("common.assignees")}</span>
            </div>
            <MemberDropdown
              value={issue.assignee_ids}
              onChange={(assignee_ids) => void update({ assignee_ids })}
              disabled={!isEditable}
              projectId={projectId}
              placeholder={t("issue.add.assignee")}
              multiple
              buttonVariant={issue.assignee_ids.length > 0 ? "transparent-without-text" : "transparent-with-text"}
              className="group w-3/5 flex-grow"
              buttonContainerClassName="w-full text-left"
              buttonClassName={`text-13 justify-between ${issue.assignee_ids.length > 0 ? "" : "text-placeholder"}`}
              hideIcon={issue.assignee_ids.length === 0}
              dropdownArrow
              dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
            />
          </div>
          <div className="flex h-8 items-center gap-2">
            <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
              <PriorityOutline className="h-4 w-4 flex-shrink-0" />
              <span>{t("common.priority")}</span>
            </div>
            <PriorityDropdown
              value={issue.priority}
              onChange={(priority) => void update({ priority })}
              disabled={!isEditable}
              buttonVariant="border-with-text"
              className="w-3/5 flex-grow rounded-sm px-2 hover:bg-layer-1"
              buttonContainerClassName="w-full text-left"
              buttonClassName="w-min h-auto whitespace-nowrap"
            />
          </div>
          <div className="flex h-8 items-center gap-2">
            <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
              <DueDateOutline className="h-4 w-4 flex-shrink-0" />
              <span>{t("common.order_by.due_date")}</span>
            </div>
            <DateDropdown
              placeholder={t("issue.add.due_date")}
              value={issue.target_date}
              onChange={(date) => void update({ target_date: date ? renderFormattedPayloadDate(date) : null })}
              minDate={minDate}
              disabled={!isEditable}
              buttonVariant="transparent-with-text"
              className="group w-3/5 flex-grow"
              buttonContainerClassName="w-full text-left"
              buttonClassName={`text-13 ${issue.target_date ? "" : "text-placeholder"}`}
              hideIcon
              clearIconClassName="h-3 w-3 hidden group-hover:inline"
            />
          </div>
          <div className="flex min-h-8 items-center gap-2">
            <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
              <LabelsOutline className="h-4 w-4 flex-shrink-0" />
              <span>{t("common.labels")}</span>
            </div>
            <div className="h-full min-h-8 w-3/5 flex-grow pt-1">
              <IssueLabel
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                disabled={!isEditable}
              />
            </div>
          </div>
        </div>
        {/* a link, not a control: never dimmed as the read-only properties are */}
        <IntakeDuplicateOf
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          duplicateToId={issue.intake?.duplicate_to_id ?? null}
        />
      </div>
    </div>
  );
});
