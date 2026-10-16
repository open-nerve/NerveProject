/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import useKeypress from "@/hooks/use-keypress";
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

export function CreateProjectModal(props: Props) {
  const { isOpen, onClose, workspaceSlug } = props;
  const { t } = useTranslation();
  // states
  const [currentStep, setCurrentStep] = useState<EProjectCreationSteps>(EProjectCreationSteps.CREATE_PROJECT);
  const [createdProjectId, setCreatedProjectId] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setCurrentStep(EProjectCreationSteps.CREATE_PROJECT);
      setCreatedProjectId(null);
    }
  }, [isOpen]);

  const handleNextStep = (projectId: string) => {
    if (!projectId) return;
    setCreatedProjectId(projectId);
    setCurrentStep(EProjectCreationSteps.FEATURE_SELECTION);
  };

  // the creation's form closes itself on Escape, but while it creates; the next step closes on it
  useKeypress("Escape", () => {
    if (isOpen && currentStep === EProjectCreationSteps.FEATURE_SELECTION) onClose();
  });

  return (
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      {/* neither step has a heading of its own (the form's top is the cover image): one hidden title names both */}
      <ModalTitle className="sr-only">{t("create_project")}</ModalTitle>
      {currentStep === EProjectCreationSteps.CREATE_PROJECT && (
        <CreateProjectForm workspaceSlug={workspaceSlug} onClose={onClose} handleNextStep={handleNextStep} />
      )}
      {currentStep === EProjectCreationSteps.FEATURE_SELECTION && (
        <ProjectFeatureUpdate projectId={createdProjectId} workspaceSlug={workspaceSlug} onClose={onClose} />
      )}
    </ModalCore>
  );
}
