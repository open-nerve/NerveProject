/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { WarningTriangleOutline } from "@makeplane/propel/icons";
// Nerve Imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { Workspace } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

type Props = {
  isOpen: boolean;
  workspace: Workspace;
  onClose: () => void;
};

/** What the form asks before the deletion: the workspace's name, and the words that confirm it. */
type TDeleteWorkspaceForm = { workspaceName: string; confirmDelete: string };

const defaultValues: TDeleteWorkspaceForm = { workspaceName: "", confirmDelete: "" };

/** The words that confirm the deletion. */
const CONFIRMATION = "delete my workspace";

export const DeleteWorkspaceModal = observer(function DeleteWorkspaceModal(props: Props) {
  const { isOpen, workspace, onClose } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { deleteWorkspace } = useWorkspace();
  const { t } = useTranslation();
  // form info
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    reset,
    watch,
  } = useForm<TDeleteWorkspaceForm>({ defaultValues });

  const confirmed = (values: TDeleteWorkspaceForm) =>
    values.workspaceName === workspace.name && values.confirmDelete === CONFIRMATION;

  const handleClose = () => {
    const timer = setTimeout(() => {
      reset(defaultValues);
      clearTimeout(timer);
    }, 350);

    onClose();
  };

  // The values submitted decide, as typed; the page follows the deletion only in the session it was sent in (M3
  // design 7.1): once another tab has moved this one to another account, the page is that account's.
  const onSubmit = (values: TDeleteWorkspaceForm) => {
    if (!confirmed(values)) return;
    return followInSession(() => deleteWorkspace(workspace), {
      done: () => {
        handleClose();
        // the root lands the caller where his workspaces, as they are now, say (M3 design 3.14)
        void navigate("/");
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_settings.settings.general.delete_modal.success_title"),
          message: t("workspace_settings.settings.general.delete_modal.success_message"),
        });
      },
      failed: (error) =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("workspace_settings.settings.general.delete_modal.error_title"),
          message: t(errorMessageKey(error)),
        }),
    });
  };

  // While the deletion is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed
  // by the dialog that sent it, and no dialog opened again offers Confirm while the request is out.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XL}
    >
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-6 p-6">
        <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
          <span
            className={cn(
              "grid size-12 shrink-0 place-items-center rounded-full bg-danger-subtle text-danger-primary sm:size-10"
            )}
          >
            <WarningTriangleOutline className="size-5 text-danger-primary" aria-hidden="true" />
          </span>
          <div>
            <div className="text-center sm:text-left">
              <h3 className="text-h5-medium">{t("workspace_settings.settings.general.delete_modal.title")}</h3>
              <p className="mt-1 text-body-xs-regular text-secondary">
                You are about to delete the workspace{" "}
                <span className="text-body-xs-semibold break-words">{workspace.name}</span>. If you confirm, you will
                lose access to all your work data in this workspace without any way to restore it. Tread very carefully.
              </p>
            </div>

            <div className="mt-4 text-secondary">
              <p className="text-body-xs-regular break-words">Type in this workspace&apos;s name to continue.</p>
              <Controller
                control={control}
                name="workspaceName"
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="workspaceName" invalid={Boolean(errors.workspaceName)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="workspaceName"
                        name="workspaceName"
                        type="text"
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder={workspace.name}
                        autoComplete="off"
                      />
                    </InputGroup>
                  </Field>
                )}
              />
            </div>

            <div className="mt-4 text-secondary">
              <p className="text-body-xs-regular">
                For final confirmation, type <span className="text-body-xs-medium text-primary">{CONFIRMATION} </span>
                below.
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
                        placeholder=""
                        autoComplete="off"
                      />
                    </InputGroup>
                  </Field>
                )}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
            {t("cancel")}
          </Button>
          <Button variant="error-fill" size="lg" type="submit" disabled={!confirmed(watch())} loading={isSubmitting}>
            {isSubmitting ? t("deleting") : t("confirm")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
});
