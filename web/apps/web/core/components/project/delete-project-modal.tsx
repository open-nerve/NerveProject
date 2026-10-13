/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Controller, useForm } from "react-hook-form";
import { WarningTriangleOutline } from "@makeplane/propel/icons";
// Nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Project } from "@nerve/api-client";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useParams, useNavigate } from "react-router";
// lib
import { followInSession } from "@/lib/in-session";

type DeleteProjectModal = {
  isOpen: boolean;
  project: Project;
  onClose: () => void;
};

/** What the form asks before the deletion: the project's name, and the words that confirm it. */
type TDeleteProjectForm = { projectName: string; confirmDelete: string };

const defaultValues: TDeleteProjectForm = {
  projectName: "",
  confirmDelete: "",
};

/** The words that confirm the deletion. */
const CONFIRMATION = "delete my project";

export function DeleteProjectModal(props: DeleteProjectModal) {
  const { isOpen, project, onClose } = props;
  // store hooks
  const { deleteProject } = useProject();
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId } = useParams();
  // form info
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    reset,
    watch,
  } = useForm<TDeleteProjectForm>({ defaultValues });

  const confirmed = (values: TDeleteProjectForm) =>
    values.projectName === project.name && values.confirmDelete === CONFIRMATION;

  const handleClose = () => {
    const timer = setTimeout(() => {
      reset(defaultValues);
      clearTimeout(timer);
    }, 350);

    onClose();
  };

  // The values submitted decide, as typed; the page follows the deletion only in the session it was sent in (M3
  // design 7.1): once another tab has moved this one to another account, the page is that account's. A page of the
  // project deleted gives way to the workspace's projects.
  const onSubmit = (values: TDeleteProjectForm) => {
    if (!workspaceSlug || !confirmed(values)) return;
    return followInSession(() => deleteProject(project), {
      done: () => {
        handleClose();
        if (projectId === project.id) void navigate(`/${workspaceSlug}/projects`);
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("project_dialogs.delete.deleted"),
        });
      },
      failed: toastRefusal,
    });
  };

  // While the deletion is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed
  // by the dialog that sent it, and no dialog opened again offers the deletion while the request is out.
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
            <h3 className="text-18 font-medium 2xl:text-20">{t("project_dialogs.delete.title")}</h3>
          </span>
        </div>
        <span>
          <p className="text-13 leading-7 text-secondary">
            <Translated
              i18nKey="project_dialogs.delete.body"
              values={{ name: <span className="font-semibold break-words">{project.name}</span> }}
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
                    placeholder={t("project_name")}
                    autoComplete="off"
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
            name="confirmDelete"
            render={({ field: { value, onChange, ref } }) => (
              <Field name="confirmDelete" invalid={Boolean(errors.confirmDelete)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="confirmDelete"
                    name="confirmDelete"
                    type="text"
                    value={value}
                    onChange={onChange}
                    ref={ref}
                    placeholder={t("project_dialogs.confirmation_placeholder", { confirmation: CONFIRMATION })}
                    autoComplete="off"
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
          <Button variant="error-fill" size="lg" type="submit" disabled={!confirmed(watch())} loading={isSubmitting}>
            {isSubmitting ? t("deleting") : t("project_dialogs.delete.title")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
