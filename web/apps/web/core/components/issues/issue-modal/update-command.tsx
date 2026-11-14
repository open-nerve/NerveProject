/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Draft } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";

import type { IssueFormData, IssueFormField, IssueFormSubmitOptions } from "./types";
import { ISSUE_FORM_FIELDS } from "./types";
import { pick } from "lodash-es";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { draftInput } from "@/services/issue/draft-input";
import type { DraftContent } from "./draft-files";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// hooks
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import type { IssueActions } from "@/hooks/use-issues-actions";
// local imports
import { CreateIssueToastActionItems } from "../create-issue-toast-action-items";

type UpdateCommandInput = {
  workspaceSlug: string | undefined;
  isDraft: boolean;
  data: IssueFormData | undefined;
  formScope: () => () => boolean;
  /** Whether saving the stored draft changes its project. */
  movesProject: (payload: IssueFormData) => boolean;
  content: DraftContent;
  updateIssue: IssueActions["updateIssue"];
  showActionItemsOnUpdate: boolean;
};

/** The modal's update of the work item or the stored draft it edits. */
export function useUpdateCommand({
  workspaceSlug,
  isDraft,
  data,
  formScope,
  movesProject,
  content,
  updateIssue,
  showActionItemsOnUpdate,
}: UpdateCommandInput) {
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const draftIssues = useWorkspaceDraftIssues();

  return async (payload: IssueFormData, options?: IssueFormSubmitOptions): Promise<Draft | undefined> => {
    if (!workspaceSlug || (!isDraft && !payload.project_id) || !data?.id) return;
    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    try {
      let response: Draft | undefined;
      if (isDraft) {
        const moving = movesProject(payload);
        if (moving) content.retire();
        try {
          response = await draftIssues.updateIssue(workspaceSlug, data.id, (held) => draftInput(payload, held.payload));
        } catch (error) {
          if (moving) content.restore();
          throw error;
        }
      } else if (updateIssue) {
        // The project moves apart; the modules are a set nerve replaces whole, never sent as the form holds it: they go
        // as the form's change, applied in the write's turn to the set the work item then holds (P7a spec 2.6, R3).
        const change: IssueChange = {
          ...pick(
            payload,
            ISSUE_FORM_FIELDS.filter(
              (field): field is Exclude<IssueFormField, "project_id" | "module_ids"> =>
                field !== "project_id" && field !== "module_ids"
            )
          ),
          modules: options?.modules,
        };
        const migration =
          options?.isDescriptionMigration &&
          Object.entries(change).every(([field, value]) => value === undefined || field === "description_html");
        await updateIssue(payload.project_id!, data.id, { ...change, ...(migration ? { skip_activity: true } : {}) });
      }
      if (!inSession()) throw new SessionChangedError();
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: t("issue_updated_successfully"),
        actionItems:
          showActionItemsOnUpdate && payload.project_id ? (
            <CreateIssueToastActionItems workspaceSlug={workspaceSlug} issueId={data.id} />
          ) : undefined,
      });
      return response;
    } catch (error) {
      if (inSession()) (options?.onRefused ?? refuse)(error);
      throw error;
    }
  };
}
