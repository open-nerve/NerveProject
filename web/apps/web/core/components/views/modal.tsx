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
// lib
import { followInSession } from "@/lib/in-session";
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

  // What follows a creation or a change (closing, going to the view, saying so) follows it only while the tab is in the
  // session it was sent in (followInSession, M3 handoff §1): after another tab moved this one to another account, the
  // page is that account's.
  const handleCreateView = (payload: IProjectView) =>
    followInSession(() => createView(workspaceSlug, projectId, payload), {
      done: (view) => {
        handleClose();
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "View created successfully.",
        });
        return navigate(`/${workspaceSlug}/projects/${projectId}/views/${view.id}`);
      },
      failed: () =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Failed to create view. Please try again.",
        }),
    });

  const handleUpdateView = (payload: IProjectView) =>
    followInSession(() => updateView(workspaceSlug, projectId, data?.id as string, payload), {
      done: handleClose,
      failed: () =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Failed to update view. Please try again.",
        }),
    });

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
      />
    </ModalCore>
  );
});
