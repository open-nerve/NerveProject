/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MouseEvent } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { generateWorkItemLink } from "@nerve/utils";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useCopyLink } from "@/hooks/use-copy-link";

type TCreateIssueToastActionItems = {
  workspaceSlug: string;
  issueId: string;
};

export const CreateIssueToastActionItems = observer(function CreateIssueToastActionItems(
  props: TCreateIssueToastActionItems
) {
  const { workspaceSlug, issueId } = props;
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  // store hooks
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getProjectIdentifierById } = useProject();

  // derived values
  const issue = getIssueById(issueId);
  const projectIdentifier = getProjectIdentifierById(issue?.project_id);

  if (!issue) return null;

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
  });

  const copyToClipboard = async (e: MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    await copyLink(workItemLink, t("common.copied_to_clipboard"));
  };

  return (
    <div className="-ml-2 flex items-center gap-1 text-11 text-secondary">
      <a
        href={workItemLink}
        target="_blank"
        rel="noopener noreferrer"
        className="rounded-sm px-2 py-1 font-medium text-accent-primary hover:bg-surface-2"
      >
        {projectIdentifier ? `${projectIdentifier}-${issue.sequence_id}` : issue.name}
      </a>

      <button
        type="button"
        className="cursor-pointer rounded-sm px-2 py-1 text-tertiary hover:bg-surface-2 hover:text-secondary"
        onClick={copyToClipboard}
      >
        {t("common.actions.copy_link")}
      </button>
    </div>
  );
});
