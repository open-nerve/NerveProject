/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, Draft, IssueDraftPayload } from "@nerve/api-client";

import type { IssueFormData, IssueDescriptionLoadState, IssueFormSubmitOptions } from "./types";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { draftInput } from "@/services/issue/draft-input";
import { ContentCopy, CopyUnconfirmedError } from "./content-copy";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// hooks
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import type { IssueActions } from "@/hooks/use-issues-actions";
// local imports
import { CreateIssueToastActionItems } from "../create-issue-toast-action-items";

type CreateCommandInput = {
  workspaceSlug: string | undefined;
  /** The seed of a seeded draft form; undefined for every other form. */
  draftSeed: IssueDraftPayload | undefined;
  copy: ContentCopy | undefined;
  formScope: () => () => boolean;
  createIssue: IssueActions["createIssue"];
  descriptionLoadState: IssueDescriptionLoadState;
};

/** The modal's create: a work item, or a draft when the form saves one. */
export function useCreateCommand({
  workspaceSlug,
  draftSeed,
  copy,
  formScope,
  createIssue,
  descriptionLoadState,
}: CreateCommandInput) {
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const draftIssues = useWorkspaceDraftIssues();

  return async (
    payload: IssueFormData,
    is_draft_issue: boolean = false,
    options?: IssueFormSubmitOptions
  ): Promise<Issue | Draft | undefined> => {
    if (!workspaceSlug || (!is_draft_issue && !payload.project_id) || descriptionLoadState !== "ready") return;
    // A seeded draft form submits only its edits; the seed holds the rest, images included.
    const input = draftInput(payload, draftSeed);
    const copying =
      copy && ContentCopy.needed(input.payload.description_html) && draftIssues.canRead(workspaceSlug)
        ? copy
        : undefined;
    if (copying?.blocked) throw new CopyUnconfirmedError();

    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    try {
      let response: Issue | Draft | undefined;
      if (copying) {
        if (!is_draft_issue && !payload.name?.trim()) return;
        response = await copying.run(input, !is_draft_issue);
      } else if (is_draft_issue) {
        response = await draftIssues.createIssue(workspaceSlug, input);
      } else if (createIssue) {
        if (!payload.name?.trim()) return;
        response = await createIssue(payload.project_id!, {
          name: payload.name,
          description_html: payload.description_html,
          ...(payload.state_id ? { state_id: payload.state_id } : {}),
          priority: payload.priority,
          assignee_ids: payload.assignee_ids,
          label_ids: payload.label_ids,
          ...(payload.start_date != null ? { start_date: payload.start_date } : {}),
          ...(payload.target_date != null ? { target_date: payload.target_date } : {}),
          ...(payload.parent_id != null ? { parent_id: payload.parent_id } : {}),
        });
      }

      if (!inSession()) throw new SessionChangedError();
      if (!response) throw new Error("Missing confirmed work item");
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: `${is_draft_issue ? t("draft_created") : t("issue_created_successfully")} `,
        actionItems: !is_draft_issue && response?.project_id && (
          <CreateIssueToastActionItems workspaceSlug={workspaceSlug} issueId={response.id} />
        ),
      });
      return response;
    } catch (error) {
      if (inSession()) (options?.onRefused ?? refuse)(error);
      throw error;
    }
  };
}
