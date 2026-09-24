/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { CycleCreateUpdateModal } from "@/components/cycles/modal";
import { CreateUpdateModuleModal } from "@/components/modules";
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
        onClose={() => toggleCreateModuleModal(false)}
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
