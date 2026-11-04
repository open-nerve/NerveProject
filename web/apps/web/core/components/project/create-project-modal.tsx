/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import useKeypress from "@/hooks/use-keypress";
import { useWorkspace } from "@/hooks/store/use-workspace";
// components
import { CreateProjectForm } from "@/components/projects/create/root";
import { ProjectFeatureUpdate } from "./project-feature-update";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  workspaceSlug: string;
};

enum EProjectCreationSteps {
  CREATE_PROJECT = "CREATE_PROJECT",
  FEATURE_SELECTION = "FEATURE_SELECTION",
}

export const CreateProjectModal = observer(function CreateProjectModal(props: Props) {
  const { isOpen, onClose, workspaceSlug } = props;
  const { t } = useTranslation();
  const workspace = useWorkspace().getWorkspaceBySlug(workspaceSlug);
  return (
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      <ModalTitle className="sr-only">{t("create_project")}</ModalTitle>
      {isOpen && <CreationSteps key={workspace?.id ?? workspaceSlug} workspaceSlug={workspaceSlug} onClose={onClose} />}
    </ModalCore>
  );
});

function CreationSteps({ workspaceSlug, onClose }: Pick<Props, "workspaceSlug" | "onClose">) {
  // states
  const [currentStep, setCurrentStep] = useState<EProjectCreationSteps>(EProjectCreationSteps.CREATE_PROJECT);
  const [createdProjectId, setCreatedProjectId] = useState<string | null>(null);

  const handleNextStep = (projectId: string) => {
    if (!projectId) return;
    setCreatedProjectId(projectId);
    setCurrentStep(EProjectCreationSteps.FEATURE_SELECTION);
  };

  // the creation's form closes itself on Escape, but while it creates; the next step closes on it
  useKeypress("Escape", () => {
    if (currentStep === EProjectCreationSteps.FEATURE_SELECTION) onClose();
  });

  return (
    <>
      {currentStep === EProjectCreationSteps.CREATE_PROJECT && (
        <CreateProjectForm workspaceSlug={workspaceSlug} onClose={onClose} handleNextStep={handleNextStep} />
      )}
      {currentStep === EProjectCreationSteps.FEATURE_SELECTION && (
        <ProjectFeatureUpdate projectId={createdProjectId} workspaceSlug={workspaceSlug} onClose={onClose} />
      )}
    </>
  );
}
