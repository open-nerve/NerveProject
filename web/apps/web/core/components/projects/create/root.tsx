/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
// components
import ProjectCommonAttributes from "@/components/project/create/common-attributes";
import ProjectCreateHeader from "@/components/project/create/header";
import ProjectCreateButtons from "@/components/project/create/project-create-buttons";
// hooks
import useKeypress from "@/hooks/use-keypress";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ProjectAttributes } from "./attributes";
import { useCreateProject, type ProjectCreationForm } from "./use-create-project";
import { getProjectFormValues } from "./utils";

export type TCreateProjectFormProps = {
  workspaceSlug: string;
  onClose: () => void;
  handleNextStep: (projectId: string) => void;
};

export const CreateProjectForm = observer(function CreateProjectForm(props: TCreateProjectFormProps) {
  const { workspaceSlug, onClose, handleNextStep } = props;
  // hooks
  const create = useCreateProject();
  // states
  const [shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier] = useState(true);
  // form info
  const methods = useForm<ProjectCreationForm>({
    defaultValues: getProjectFormValues(),
    reValidateMode: "onChange",
  });
  const {
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { isSubmitting },
  } = methods;
  const { isMobile } = usePlatformOS();

  // The form is busy until nerve has answered and the page has followed: created, the modal's next step.
  const onSubmit = async (values: ProjectCreationForm) => {
    const project = await create(workspaceSlug, values, setError);
    if (project) handleNextStep(project.id);
  };

  const handleClose = () => {
    onClose();
    setShouldAutoSyncIdentifier(true);
    setTimeout(() => {
      reset();
    }, 300);
  };

  // While the creation is out the form cannot be closed (Escape, its close button, Cancel): nerve's answer is followed
  // by the form that sent it.
  useKeypress("Escape", () => {
    if (!isSubmitting) handleClose();
  });

  return (
    <FormProvider {...methods}>
      <ProjectCreateHeader handleClose={handleClose} isMobile={isMobile} />

      <form onSubmit={handleSubmit(onSubmit)} className="px-3">
        <div className="mt-9 space-y-6 pb-5">
          <ProjectCommonAttributes
            setValue={setValue}
            isMobile={isMobile}
            shouldAutoSyncIdentifier={shouldAutoSyncIdentifier}
            setShouldAutoSyncIdentifier={setShouldAutoSyncIdentifier}
          />
          <ProjectAttributes isMobile={isMobile} />
        </div>
        <ProjectCreateButtons handleClose={handleClose} />
      </form>
    </FormProvider>
  );
});
