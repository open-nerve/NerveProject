/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect } from "react";
import { observer } from "mobx-react";
// components
import { CycleCreateUpdateModal } from "@/components/cycles/modal";
import { CreateUpdateModuleModal } from "@/components/modules/modal";
import { CreateUpdateProjectViewModal } from "@/components/views/modal";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";

export type TProjectLevelModalsProps = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectLevelModals = observer(function ProjectLevelModals(props: TProjectLevelModalsProps) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const {
    isCreateCycleModalOpen,
    toggleCreateCycleModal,
    isCreateModuleModalOpen,
    toggleCreateModuleModal,
    isCreateViewModalOpen,
    toggleCreateViewModal,
  } = useCommandPalette();
  // The open flags of the dialogs this host renders (a new cycle's, module's and view's) are the command palette
  // store's, which outlives the project; the host is keyed by its project where it is mounted (ProjectsAppPowerKProvider)
  // and closes them as it unmounts, before the next project paints, so that a switch, a Back or Forward too, never opens
  // one project's dialog in another (v0 design 7.7, state keyed by id).
  useLayoutEffect(
    () => () => {
      toggleCreateCycleModal(false);
      toggleCreateModuleModal(false);
      toggleCreateViewModal(false);
    },
    [toggleCreateCycleModal, toggleCreateModuleModal, toggleCreateViewModal]
  );

  return (
    <>
      <CycleCreateUpdateModal
        isOpen={isCreateCycleModalOpen}
        handleClose={() => toggleCreateCycleModal(false)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
      <CreateUpdateModuleModal
        isOpen={isCreateModuleModalOpen}
        handleClose={() => toggleCreateModuleModal(false)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
      <CreateUpdateProjectViewModal
        isOpen={isCreateViewModalOpen}
        onClose={() => toggleCreateViewModal(false)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
    </>
  );
});
