/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams, useNavigate } from "react-router";
// types
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IProjectView } from "@nerve/types";
// ui
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useProjectView } from "@/hooks/store/use-project-view";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  data: IProjectView;
  isOpen: boolean;
  onClose: () => void;
};

export const DeleteProjectViewModal = observer(function DeleteProjectViewModal(props: Props) {
  const { data, isOpen, onClose } = props;
  // states
  const [isDeleteLoading, setIsDeleteLoading] = useState(false);
  // router
  const { workspaceSlug, projectId } = useParams();
  const navigate = useNavigate();
  // store hooks
  const { deleteView } = useProjectView();
  const { t } = useTranslation();
  const handleClose = () => {
    onClose();
    setIsDeleteLoading(false);
  };

  // What follows the deletion (closing, going to the project's views, saying so) follows it only while the tab is in
  // the session it was sent in (followInSession, M3 handoff §1): after another tab moved this one to another account,
  // the page is that account's.
  const handleDeleteView = async () => {
    if (!workspaceSlug || !projectId) return;
    setIsDeleteLoading(true);
    await followInSession(() => deleteView(workspaceSlug, projectId, data.id), {
      done: () => {
        handleClose();
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "View deleted successfully.",
        });
        return navigate(`/${workspaceSlug}/projects/${projectId}/views`);
      },
      failed: () =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "View could not be deleted. Please try again.",
        }),
    });
    setIsDeleteLoading(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeleteView}
      isSubmitting={isDeleteLoading}
      isOpen={isOpen}
      title={t("project_views.delete_view.title")}
      content={<>{t("project_views.delete_view.content")}</>}
    />
  );
});
