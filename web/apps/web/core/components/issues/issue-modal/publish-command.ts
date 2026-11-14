/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormData, IssueDescriptionLoadState } from "./types";
import { useState } from "react";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { sessionGuard } from "@/lib/in-session";
import { draftInput } from "@/services/issue/draft-input";
import type { DraftContent } from "./draft-files";
import { problemCode } from "@/lib/api-error";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// hooks
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";

type PublishCommandInput = {
  workspaceSlug: string | undefined;
  isDraft: boolean;
  data: IssueFormData | undefined;
  descriptionLoadState: IssueDescriptionLoadState;
  formScope: () => () => boolean;
  /** Throws while the draft's transfers are on the wire; the user finishes or cancels them first. */
  assertSettled: () => void;
  content: DraftContent;
  finishClose: () => void;
};

/** The modal's publish of the stored draft it edits, and whether nerve refused it for files not yet uploaded. */
export function usePublishCommand({
  workspaceSlug,
  isDraft,
  data,
  descriptionLoadState,
  formScope,
  assertSettled,
  content,
  finishClose,
}: PublishCommandInput) {
  const [publishBlocked, setPublishBlocked] = useState(false);
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const draftIssues = useWorkspaceDraftIssues();

  const executePublish = async (payload: IssueFormData) => {
    if (!workspaceSlug || !isDraft || !data?.id || descriptionLoadState !== "ready") return;
    assertSettled();
    const current = formScope();
    const inSession = sessionGuard();
    setPublishBlocked(false);
    // Publishing moves every draft file on the server; this lifetime ends before the save that precedes it.
    content.retire();
    try {
      await draftIssues.moveIssue(workspaceSlug, data.id, (held) => draftInput(payload, held.payload));
      if (!inSession() || !current()) return;
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("issue_created_successfully") });
      finishClose();
    } catch (error) {
      content.restore();
      if (inSession() && current()) {
        if (problemCode(error) === "asset.not_uploaded") setPublishBlocked(true);
        refuse(error);
      }
      throw error;
    }
  };

  return { executePublish, publishBlocked };
}
