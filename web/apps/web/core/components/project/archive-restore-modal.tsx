/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// ui
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useNavigate } from "react-router";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  workspaceSlug: string;

  projectId: string;
  isOpen: boolean;
  onClose: () => void;
  archive: boolean;
};

export function ArchiveRestoreProjectModal(props: Props) {
  const { workspaceSlug, projectId, isOpen, onClose, archive } = props;
  // router
  const navigate = useNavigate();
  // states
  const [isLoading, setIsLoading] = useState(false);
  // store hooks
  const { getProjectById, archiveProject, restoreProject } = useProject();
  const toastRefusal = useRefusalToast();

  const projectDetails = getProjectById(projectId);
  if (!projectDetails) return null;

  // The page follows the change only in the session it was sent in (M3 design 7.1): it says so, and gives way to the
  // workspace's projects; busy until it has. A refusal is said by nerve's reason, and the dialog stays.
  const handleChange = async () => {
    setIsLoading(true);
    await followInSession(() => (archive ? archiveProject(projectId) : restoreProject(projectId)), {
      done: () => {
        setToast(
          archive
            ? {
                type: TOAST_TYPE.SUCCESS,
                title: "Archive success",
                message: `${projectDetails.name} has been archived successfully`,
              }
            : {
                type: TOAST_TYPE.SUCCESS,
                title: "Restore success",
                message: `You can find ${projectDetails.name} in your projects.`,
              }
        );
        onClose();
        return navigate(`/${workspaceSlug}/projects`);
      },
      failed: toastRefusal,
    });
    setIsLoading(false);
  };

  // While the change is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed
  // by the dialog that sent it.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isLoading ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.LG}
    >
      <div className="px-5 py-4">
        <h3 className="text-18 font-medium 2xl:text-20">
          {archive ? "Archive" : "Restore"} {projectDetails.name}
        </h3>
        <p className="mt-3 text-13 text-secondary">
          {archive
            ? "This project and its work items, cycles and modules will be archived. Its work items won't appear in search. Only project admins can restore the project."
            : "Restoring a project will activate it and make it visible to all members of the project. Are you sure you want to continue?"}
        </p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" onClick={() => void handleChange()} loading={isLoading}>
            {archive ? (isLoading ? "Archiving" : "Archive") : isLoading ? "Restoring" : "Restore"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
