/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// types
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IProjectView } from "@nerve/types";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useProjectView } from "@/hooks/store/use-project-view";
import { useNavigate } from "react-router";
import useKeypress from "@/hooks/use-keypress";
// local imports
import { ProjectViewForm } from "./form";

type Props = {
  data?: IProjectView | null;
  isOpen: boolean;
  onClose: () => void;
  preLoadedData?: Partial<IProjectView> | null;
  workspaceSlug: string;
  projectId: string;
};

export const CreateUpdateProjectViewModal = observer(function CreateUpdateProjectViewModal(props: Props) {
  const { data, isOpen, onClose, preLoadedData, workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { createView, updateView } = useProjectView();

  const handleClose = () => {
    onClose();
  };

  const handleCreateView = async (payload: IProjectView) => {
    try {
      const res = await createView(workspaceSlug, projectId, payload);
      handleClose();
      navigate(`/${workspaceSlug}/projects/${projectId}/views/${res.id}`);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "View created successfully.",
      });
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to create view. Please try again.",
      });
    }
  };

  const handleUpdateView = async (payload: IProjectView) => {
    try {
      await updateView(workspaceSlug, projectId, data?.id as string, payload);

      handleClose();
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to update view. Please try again.",
      });
    }
  };

  const handleFormSubmit = async (formData: IProjectView) => {
    if (!data) await handleCreateView(formData);
    else await handleUpdateView(formData);
  };

  useKeypress("Escape", () => {
    if (isOpen) handleClose();
  });

  return (
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <ProjectViewForm
        data={data}
        handleClose={handleClose}
        handleFormSubmit={handleFormSubmit}
        preLoadedData={preLoadedData}
        projectId={projectId}
      />
    </ModalCore>
  );
});
