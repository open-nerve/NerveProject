/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// components
import ProjectCommonAttributes from "@/components/project/create/common-attributes";
import ProjectCreateHeader from "@/components/project/create/header";
import ProjectCreateButtons from "@/components/project/create/project-create-buttons";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { usePlatformOS } from "@/hooks/use-platform-os";
// nerve imports
import type { ProjectCreate } from "@nerve/api-client";
import { ProjectAttributes } from "./attributes";
import { getProjectFormValues } from "./utils";

export type TCreateProjectFormProps = {
  workspaceSlug: string;
  onClose: () => void;
  handleNextStep: (projectId: string) => void;
};

export const CreateProjectForm = observer(function CreateProjectForm(props: TCreateProjectFormProps) {
  const { workspaceSlug, onClose, handleNextStep } = props;
  // store
  const { t } = useTranslation();
  const { createProject } = useProject();
  // states
  const [shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier] = useState(true);
  // form info
  const methods = useForm<ProjectCreate>({
    defaultValues: getProjectFormValues(),
    reValidateMode: "onChange",
  });
  const { handleSubmit, reset, setValue } = methods;
  const { isMobile } = usePlatformOS();

  const onSubmit = async (formData: ProjectCreate) => {
    // Upper case identifier
    return createProject(workspaceSlug, { ...formData, identifier: formData.identifier.toUpperCase() })
      .then((res) => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("success"),
          message: t("project_created_successfully"),
        });
        return handleNextStep(res.id);
      })
      .catch((err) => {
        try {
          // Handle the new error format where codes are nested in arrays under field names
          const errorData = err?.data ?? {};

          const nameError = errorData.name?.includes("PROJECT_NAME_ALREADY_EXIST");
          const identifierError = errorData?.identifier?.includes("PROJECT_IDENTIFIER_ALREADY_EXIST");
          const nameSpecialCharError = errorData?.name?.includes("PROJECT_NAME_CANNOT_CONTAIN_SPECIAL_CHARACTERS");

          if (nameError || identifierError || nameSpecialCharError) {
            if (nameError) {
              setToast({
                type: TOAST_TYPE.ERROR,
                title: t("toast.error"),
                message: t("project_name_already_taken"),
              });
            }

            if (identifierError) {
              setToast({
                type: TOAST_TYPE.ERROR,
                title: t("toast.error"),
                message: t("project_identifier_already_taken"),
              });
            }

            if (nameSpecialCharError) {
              setToast({
                type: TOAST_TYPE.ERROR,
                title: t("toast.error"),
                message: t("project_name_cannot_contain_special_characters"),
              });
            }
          } else {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: t("toast.error"),
              message: t("something_went_wrong"),
            });
          }
        } catch (error) {
          // Fallback error handling if the error processing fails
          console.error("Error processing API error:", error);
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("something_went_wrong"),
          });
        }
      });
  };

  const handleClose = () => {
    onClose();
    setShouldAutoSyncIdentifier(true);
    setTimeout(() => {
      reset();
    }, 300);
  };

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
