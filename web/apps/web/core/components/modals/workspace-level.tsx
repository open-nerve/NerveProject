/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect } from "react";
import { observer } from "mobx-react";
// components
import { CreateProjectModal } from "@/components/project/create-project-modal";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";

export type TWorkspaceLevelModalsProps = {
  workspaceSlug: string;
};

export const WorkspaceLevelModals = observer(function WorkspaceLevelModals(props: TWorkspaceLevelModalsProps) {
  const { workspaceSlug } = props;
  // store hooks
  const { isCreateProjectModalOpen, toggleCreateProjectModal } = useCommandPalette();
  // The open flags of the dialogs this host renders (the project's creation, isCreateProjectModalOpen) are the command
  // palette store's, which outlives the workspace's keyed content (v0 design 7.7, state keyed by id): the host closes
  // them as it unmounts with that content, before the next workspace paints, so a switch, a Back or Forward too, never
  // opens one workspace's dialog in another.
  useLayoutEffect(() => () => toggleCreateProjectModal(false), [toggleCreateProjectModal]);

  return (
    <>
      <CreateProjectModal
        isOpen={isCreateProjectModalOpen}
        onClose={() => toggleCreateProjectModal(false)}
        workspaceSlug={workspaceSlug}
      />
    </>
  );
});
