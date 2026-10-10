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
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Project } from "@nerve/api-client";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
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
          title: "Success!",
          message: "Project deleted successfully.",
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
            <h3 className="text-18 font-medium 2xl:text-20">Delete project</h3>
          </span>
        </div>
        <span>
          <p className="text-13 leading-7 text-secondary">
            Are you sure you want to delete project <span className="font-semibold break-words">{project.name}</span>?
            All of the data related to the project will be permanently removed. This action cannot be undone
          </p>
        </span>
        <div className="text-secondary">
          <p className="text-13 break-words">
            Enter the project name <span className="font-medium text-primary">{project.name}</span> to continue:
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
                    placeholder="Project name"
                    autoComplete="off"
                  />
                </InputGroup>
              </Field>
            )}
          />
        </div>
        <div className="text-secondary">
          <p className="text-13">
            To confirm, type <span className="font-medium text-primary">{CONFIRMATION}</span> below:
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
                    placeholder="Enter 'delete my project'"
                    autoComplete="off"
                  />
                </InputGroup>
              </Field>
            )}
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button variant="error-fill" size="lg" type="submit" disabled={!confirmed(watch())} loading={isSubmitting}>
            {isSubmitting ? "Deleting" : "Delete project"}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
