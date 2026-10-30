/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import { useMemo, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useLabelChanges, type IssueLabelCreation } from "@/components/labels/use-label-changes";
import { useDetailCommand } from "../use-issue-operations";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";

// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProjectInbox } from "@/hooks/store/use-project-inbox";
// types
import { LabelList, IssueLabelSelectRoot } from "./";
// TODO: Fix this import statement, as core should not import from ee

export type TIssueLabel = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  isInboxIssue?: boolean;
  onLabelUpdate?: (labelIds: string[]) => void | Promise<void>;
};

export type TLabelOperations = {
  updateIssue: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<IssueDetail>) => Promise<void>;
  create: IssueLabelCreation;
};

export function IssueLabel(props: TIssueLabel) {
  return <IssueLabelContent key={JSON.stringify([props.workspaceSlug, props.projectId, props.issueId])} {...props} />;
}
const IssueLabelContent = observer(function IssueLabelContent(props: TIssueLabel) {
  const { workspaceSlug, projectId, issueId, disabled = false, isInboxIssue = false, onLabelUpdate } = props;
  const pending = useRef(false);
  const [submitting, setSubmitting] = useState(false);
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const { create } = useLabelChanges(capture);
  const run = useDetailCommand(workspaceSlug, projectId, issueId);
  // hooks
  const { updateIssue } = useIssueDetail();
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getIssueInboxByIssueId } = useProjectInbox();

  const issue = isInboxIssue ? getIssueInboxByIssueId(issueId)?.issue : getIssueById(issueId);

  const labelOperations: TLabelOperations = useMemo(
    () => ({
      updateIssue: async (slug: string, issueProjectId: string, workItemId: string, data: Partial<IssueDetail>) => {
        if (pending.current) return;
        pending.current = true;
        setSubmitting(true);
        try {
          await run(async () => {
            if (onLabelUpdate) await onLabelUpdate(data.label_ids ?? []);
            else await updateIssue(slug, issueProjectId, workItemId, data);
          });
        } finally {
          pending.current = false;
          setSubmitting(false);
        }
      },
      create: (data, underField, done) => create(projectId, data, underField, done),
    }),
    [updateIssue, run, create, onLabelUpdate, projectId]
  );

  return (
    <div className="relative flex min-h-7.5 w-full flex-wrap items-center gap-1 px-2">
      <LabelList
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        values={issue?.label_ids || []}
        labelOperations={labelOperations}
        disabled={disabled || submitting}
      />

      {!disabled && (
        <IssueLabelSelectRoot
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={submitting}
          showCreationToast={!isInboxIssue}
          values={issue?.label_ids || []}
          labelOperations={labelOperations}
        />
      )}
    </div>
  );
});
