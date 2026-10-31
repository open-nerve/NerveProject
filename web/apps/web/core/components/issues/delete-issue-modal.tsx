/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useEffect, useState, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// types
import { PROJECT_ERROR_MESSAGES, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TDeDupeIssue } from "@nerve/types";
// ui
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "./issue-detail-widgets/use-detail-scope";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  dataId?: string | null | undefined;
  data?: Issue | TDeDupeIssue;
  isSubIssue?: boolean;
  /** Deletes it: the confirmation is busy until it settles. */
  onSubmit: () => Promise<void>;
};

export const DeleteIssueModal = observer(function DeleteIssueModal(props: Props) {
  const { dataId, data, isOpen, handleClose, isSubIssue = false, onSubmit } = props;
  // states
  const [isDeleting, setIsDeleting] = useState(false);
  const sending = useRef(false);
  // store hooks
  const { workspaceSlug } = useParams();
  const { issueMap } = useIssues();
  const { getProjectById } = useProject();
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();

  const { data: currentUser } = useUser();
  const issue = data ? data : dataId ? issueMap[dataId] : undefined;
  const capture = useDetailScope(workspaceSlug ?? "", issue?.project_id ?? undefined, `${issue?.id ?? ""}:${isOpen}`);
  const refusal = useRefusalToast();

  useEffect(() => {
    setIsDeleting(false);
    sending.current = false;
  }, [isOpen, issue?.id]);

  if (!dataId && !data) return null;

  // derived values
  const projectDetails = getProjectById(issue?.project_id);
  const issueCreatorId = issue && ("created_by_id" in issue ? issue.created_by_id : issue.created_by);
  const isIssueCreator = !!issueCreatorId && issueCreatorId === currentUser?.id;

  const canPerformProjectAdminActions = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectDetails?.id
  );

  const authorized = isIssueCreator || canPerformProjectAdminActions;

  const finishClose = () => {
    setIsDeleting(false);
    handleClose();
  };
  const onClose = () => {
    if (!sending.current) finishClose();
  };

  const handleIssueDelete = async () => {
    const current = capture();
    if (!current() || sending.current) return;
    sending.current = true;
    setIsDeleting(true);

    if (!authorized) {
      setToast({
        title: t(PROJECT_ERROR_MESSAGES.permissionError.i18n_title),
        type: TOAST_TYPE.ERROR,
        message:
          PROJECT_ERROR_MESSAGES.permissionError.i18n_message && t(PROJECT_ERROR_MESSAGES.permissionError.i18n_message),
      });
      setIsDeleting(false);
      sending.current = false;
      return;
    }
    await followInSession(onSubmit, {
      done: () => {
        if (!current()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("common.success"),
          message: t("entity.delete.success", {
            entity: isSubIssue ? t("common.sub_work_item") : t("common.work_item"),
          }),
        });
        finishClose();
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError)) refusal(error);
      },
    });
    if (current()) {
      sending.current = false;
      setIsDeleting(false);
    }
  };

  return (
    <AlertModalCore
      handleClose={onClose}
      handleSubmit={handleIssueDelete}
      isSubmitting={isDeleting}
      isOpen={isOpen}
      title={t("entity.delete.label", { entity: t("common.work_item") })}
      primaryButtonText={{ loading: t("deleting"), default: t("delete") }}
      secondaryButtonText={t("common.cancel")}
      content={t("delete_issue_subtree", {
        identifier:
          projectDetails?.identifier && issue
            ? `${projectDetails.identifier}-${issue.sequence_id}`
            : t("common.work_item"),
      })}
    />
  );
});
