/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useNavigate } from "react-router";
// types
import { Button } from "@nerve/propel/button";
import type { Project } from "@nerve/api-client";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// local imports
import { useJoinProject } from "./use-join-project";

// type
type TJoinProjectModalProps = {
  isOpen: boolean;
  workspaceSlug: string;
  project: Project;
  handleClose: () => void;
};

export function JoinProjectModal(props: TJoinProjectModalProps) {
  const { handleClose, isOpen, project, workspaceSlug } = props;
  // states
  const [isJoiningLoading, setIsJoiningLoading] = useState(false);
  // hooks
  const join = useJoinProject();
  // router
  const navigate = useNavigate();

  // One join at a time: the dialog is busy, its button loading, until nerve has answered and the page has followed
  // (useJoinProject); joined, the dialog closes and the page opens the project.
  const handleJoin = async () => {
    setIsJoiningLoading(true);
    await join(project.id, async () => {
      handleClose();
      await navigate(`/${workspaceSlug}/projects/${project.id}/issues`);
    });
    setIsJoiningLoading(false);
  };

  // While the join is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed by
  // the dialog that sent it.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isJoiningLoading ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XL}
    >
      <div className="space-y-5 px-5 py-8 sm:p-6">
        <h3 className="text-16 leading-6 font-medium text-primary">Join Project?</h3>
        <p>
          Are you sure you want to join the project <span className="font-semibold break-words">{project?.name}</span>?
          Please click the &apos;Join Project&apos; button below to continue.
        </p>
        <div className="space-y-3" />
      </div>
      <div className="mt-5 flex justify-end gap-2 px-5 pb-8 sm:px-6 sm:pb-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isJoiningLoading}>
          Cancel
        </Button>
        <Button variant="primary" size="lg" type="submit" onClick={handleJoin} loading={isJoiningLoading}>
          {isJoiningLoading ? "Joining..." : "Join Project"}
        </Button>
      </div>
    </ModalCore>
  );
}
