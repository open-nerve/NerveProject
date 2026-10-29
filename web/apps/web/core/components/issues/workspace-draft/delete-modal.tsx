/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import type { Draft } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { AlertModalCore } from "@nerve/ui";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "../issue-detail-widgets/use-detail-scope";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  workspaceSlug: string;
  data: Draft;
  onSubmit: () => Promise<void>;
};
export const WorkspaceDraftIssueDeleteIssueModal = observer(function WorkspaceDraftIssueDeleteIssueModal({
  isOpen,
  handleClose,
  workspaceSlug,
  data,
  onSubmit,
}: Props) {
  const [isDeleting, setIsDeleting] = useState(false);
  const busy = useRef(false);
  const { canRead } = useWorkspaceDraftIssues();
  const scope = useDetailScope(workspaceSlug, undefined, data.id, true);
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const close = () => {
    if (!busy.current) handleClose();
  };
  const submit = async () => {
    const current = scope();
    if (busy.current || !canRead(workspaceSlug) || !current()) return;
    busy.current = true;
    setIsDeleting(true);
    try {
      await onSubmit();
      if (!current()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: t("workspace_draft_issues.toasts.deleted.success"),
      });
      handleClose();
    } catch (error) {
      if (current()) refuse(error);
    } finally {
      busy.current = false;
      setIsDeleting(false);
    }
  };
  if (!canRead(workspaceSlug)) return null;
  return (
    <AlertModalCore
      handleClose={close}
      handleSubmit={submit}
      isSubmitting={isDeleting}
      isOpen={isOpen}
      title={t("workspace_draft_issues.delete_modal.title")}
      content={<>{t("workspace_draft_issues.delete_modal.description")}</>}
      primaryButtonText={{ loading: t("deleting"), default: t("delete") }}
      secondaryButtonText={t("cancel")}
    />
  );
});
