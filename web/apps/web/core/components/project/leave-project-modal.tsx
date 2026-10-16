/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { WarningTriangleOutline } from "@makeplane/propel/icons";
// Nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Project } from "@nerve/api-client";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useParams } from "react-router";
// components
import { Translated } from "@/components/common/translated";
import { useProjectMembershipChanges } from "@/components/project/settings/use-project-membership-changes";

type FormData = {
  projectName: string;
  confirmLeave: string;
};

const defaultValues: FormData = {
  projectName: "",
  confirmLeave: "",
};

/** The words that confirm the leaving. */
const CONFIRMATION = "Leave Project";

export interface ILeaveProjectModal {
  project: Project;
  isOpen: boolean;
  onClose: () => void;
}

export const LeaveProjectModal = observer(function LeaveProjectModal(props: ILeaveProjectModal) {
  const { project, isOpen, onClose } = props;
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { leave } = useProjectMembershipChanges(workspaceSlug ?? "", project.id);
  const { t } = useTranslation();

  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    reset,
  } = useForm({ defaultValues });

  const handleClose = () => {
    reset({ ...defaultValues });
    onClose();
  };

  const onSubmit = async (data: FormData) => {
    if (!workspaceSlug) return;

    if (data.projectName !== project.name) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t("project_dialogs.leave.name_mismatch"),
      });
      return;
    }
    if (data.confirmLeave !== CONFIRMATION) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t("project_dialogs.leave.confirmation_mismatch", { confirmation: CONFIRMATION }),
      });
      return;
    }
    // the leaving first: once nerve has made it the modal closes and the workspace's projects show; one it refuses
    // leaves the modal open
    await leave(handleClose);
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-6 p-6">
        <div className="flex w-full items-center justify-start gap-6">
          <span className="place-items-center rounded-full bg-danger-subtle p-4">
            <WarningTriangleOutline className="h-6 w-6 text-danger-primary" aria-hidden="true" />
          </span>
          <span className="flex items-center justify-start">
            <ModalTitle className="text-18 font-medium 2xl:text-20">{t("project_dialogs.leave.title")}</ModalTitle>
          </span>
        </div>

        <span>
          <p className="text-13 leading-7 text-secondary">
            <Translated
              i18nKey="project_dialogs.leave.body"
              values={{ name: <span className="font-medium text-primary">{project.name}</span> }}
            />
          </p>
        </span>

        <div className="text-secondary">
          <p className="text-13 break-words">
            <Translated
              i18nKey="project_dialogs.type_project_name"
              values={{ name: <span className="font-medium text-primary">{project.name}</span> }}
            />
          </p>
          <Controller
            control={control}
            name="projectName"
            rules={{ required: true }}
            render={({ field: { value, onChange, ref } }) => (
              <Field name="projectName" invalid={Boolean(errors.projectName)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="projectName"
                    name="projectName"
                    type="text"
                    value={value}
                    onChange={onChange}
                    ref={ref}
                    placeholder={t("project_dialogs.leave.name_placeholder")}
                  />
                </InputGroup>
              </Field>
            )}
          />
        </div>

        <div className="text-secondary">
          <p className="text-13">
            <Translated
              i18nKey="project_dialogs.type_confirmation"
              values={{ confirmation: <span className="font-medium text-primary">{CONFIRMATION}</span> }}
            />
          </p>
          <Controller
            control={control}
            name="confirmLeave"
            render={({ field: { value, onChange, ref } }) => (
              <Field name="confirmLeave" invalid={Boolean(errors.confirmLeave)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="confirmLeave"
                    name="confirmLeave"
                    type="text"
                    value={value}
                    onChange={onChange}
                    ref={ref}
                    placeholder={t("project_dialogs.confirmation_placeholder", { confirmation: CONFIRMATION })}
                  />
                </InputGroup>
              </Field>
            )}
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
            {t("cancel")}
          </Button>
          <Button variant="error-fill" size="lg" type="submit" loading={isSubmitting}>
            {isSubmitting ? t("leaving") : t("project_dialogs.leave.title")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
});
