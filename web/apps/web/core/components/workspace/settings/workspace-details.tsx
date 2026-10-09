/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { useParams } from "react-router";
// Nerve Imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { ORGANIZATION_SIZE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { OrganizationSize, Workspace, WorkspaceUpdate } from "@nerve/api-client";
import { CustomSelect } from "@nerve/ui";
import { cn, copyUrlToClipboard, getFileURL, validateWorkspaceName } from "@nerve/utils";
// components
import { TimezoneSelect } from "@/components/global/timezone-select";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
// components
import { DeleteWorkspaceSection } from "@/components/workspace/delete-workspace-section";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * The form's values: what an admin may change of the workspace. The size is null while the workspace has none, which
 * nerve's WorkspaceUpdate cannot set: the select holds a value from the start (undefined would leave it uncontrolled).
 */
type TWorkspaceForm = Required<Pick<WorkspaceUpdate, "name" | "timezone">> & {
  organization_size: OrganizationSize | null;
};

/** The form's values for a workspace; a size the form does not offer (none was given) shows as none. */
const formValues = (workspace: Workspace): TWorkspaceForm => ({
  name: workspace.name,
  organization_size: ORGANIZATION_SIZE.find((size) => size === workspace.organization_size) ?? null,
  timezone: workspace.timezone,
});

export const WorkspaceDetails = observer(function WorkspaceDetails() {
  // states
  const [isLoading, setIsLoading] = useState(false);
  // router: the address's workspace, as the caller's list has it (the wrapper shows the page once it has)
  const { workspaceSlug } = useParams();
  // store hooks
  const { getWorkspaceBySlug, updateWorkspace } = useWorkspace();
  const workspace = workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();

  // form info
  const {
    handleSubmit,
    control,
    reset,
    watch,
    formState: { errors },
  } = useForm<TWorkspaceForm>({
    defaultValues: workspace ? formValues(workspace) : { name: "", organization_size: "2-10", timezone: "UTC" },
  });

  // the fields the form edits, which nerve's WorkspaceUpdate takes; the page follows the change only in the session it
  // was sent in (M3 design 7.1)
  const onSubmit = async (formData: TWorkspaceForm) => {
    if (!workspace) return;

    setIsLoading(true);

    const payload: WorkspaceUpdate = {
      name: formData.name,
      organization_size: formData.organization_size ?? undefined,
      timezone: formData.timezone,
    };

    await followInSession(() => updateWorkspace(workspace.slug, payload), {
      done: () =>
        setToast({
          title: "Success!",
          type: TOAST_TYPE.SUCCESS,
          message: "Workspace updated successfully",
        }),
      failed: (error) =>
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) }),
    });
    setIsLoading(false);
  };

  const handleCopyUrl = () => {
    if (!workspace) return;

    void copyUrlToClipboard(`${workspace.slug}`)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Workspace URL copied to the clipboard.",
        });
        return undefined;
      })
      // the browser did not let the page write the clipboard
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: t("something_went_wrong_please_try_again"),
        })
      );
  };

  useEffect(() => {
    if (workspace) reset(formValues(workspace));
  }, [workspace, reset]);

  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);

  if (!workspace) return null;

  return (
    <>
      <div className={cn("flex w-full flex-col gap-y-7", { "opacity-60": !isAdmin })}>
        <div className="flex items-center gap-5">
          <div className="flex shrink-0 flex-col gap-1">
            {workspace.logo_url ? (
              <div className="relative flex size-14">
                <img
                  src={getFileURL(workspace.logo_url)}
                  className="absolute top-0 left-0 size-full rounded-md object-cover"
                  alt="Workspace Logo"
                />
              </div>
            ) : (
              <div className="relative grid size-14 place-items-center rounded-md bg-accent-primary text-24 text-on-color uppercase">
                {workspace.name.charAt(0)}
              </div>
            )}
          </div>
          <div className="flex flex-col gap-1">
            <div className="mb:-my-5 text-h5-semibold leading-6">{watch("name")}</div>
            <button type="button" onClick={handleCopyUrl} className="text-left text-body-xs-regular tracking-tight">{`${
              typeof window !== "undefined" && window.location.origin.replace("http://", "").replace("https://", "")
            }/${workspace.slug}`}</button>
          </div>
        </div>
        <div className="flex flex-col gap-7">
          <div className="grid-col grid w-full grid-cols-1 items-center justify-between gap-10 xl:grid-cols-2 2xl:grid-cols-3">
            <div className="flex flex-col gap-2">
              <h4 className="text-body-sm-medium text-tertiary">{t("workspace_settings.settings.general.name")}</h4>
              <Controller
                control={control}
                name="name"
                rules={{
                  validate: (value) => validateWorkspaceName(value, true),
                }}
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="name" invalid={Boolean(errors.name)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="name"
                        name="name"
                        type="text"
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder={t("workspace_settings.settings.general.name")}
                        disabled={!isAdmin}
                      />
                    </InputGroup>
                  </Field>
                )}
              />
              {errors.name && <p className="text-caption-sm-regular text-danger-primary">{errors.name.message}</p>}
            </div>
            <div className="flex flex-col gap-2">
              <h4 className="text-body-sm-medium text-tertiary">
                {t("workspace_settings.settings.general.company_size")}
              </h4>
              <Controller
                name="organization_size"
                control={control}
                render={({ field: { value, onChange } }) => (
                  <CustomSelect
                    value={value}
                    onChange={onChange}
                    label={
                      ORGANIZATION_SIZE.find((c) => c === value) ??
                      t("workspace_settings.settings.general.errors.company_size.select_a_range")
                    }
                    buttonClassName="border border-subtle bg-layer-2 !shadow-none !rounded-md"
                    input
                    disabled={!isAdmin}
                  >
                    {ORGANIZATION_SIZE.map((item) => (
                      <CustomSelect.Option key={item} value={item}>
                        {item}
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                )}
              />
            </div>
            <div className="flex flex-col gap-2">
              <h4 className="text-body-sm-medium text-tertiary">{t("workspace_settings.settings.general.url")}</h4>
              <Field name="url">
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="url"
                    name="url"
                    type="url"
                    value={`${
                      typeof window !== "undefined" &&
                      window.location.origin.replace("http://", "").replace("https://", "")
                    }/${workspace.slug}`}
                    readOnly
                    disabled
                  />
                </InputGroup>
              </Field>
            </div>
            <div className="flex flex-col gap-2">
              <h4 className="text-body-sm-medium text-tertiary">
                {t("workspace_settings.settings.general.workspace_timezone")}
              </h4>
              <Controller
                name="timezone"
                control={control}
                render={({ field: { value, onChange } }) => (
                  <>
                    <TimezoneSelect value={value} onChange={onChange} disabled={!isAdmin} />
                  </>
                )}
              />
            </div>
          </div>
        </div>
        {isAdmin && (
          <div className="flex items-center justify-between py-2">
            <Button
              variant="primary"
              size="lg"
              onClick={(e) => {
                void handleSubmit(onSubmit)(e);
              }}
              loading={isLoading}
            >
              {isLoading ? t("updating") : t("workspace_settings.settings.general.update_workspace")}
            </Button>
          </div>
        )}
      </div>
      {isAdmin && (
        <div className="mt-10">
          <DeleteWorkspaceSection workspace={workspace} />
        </div>
      )}
    </>
  );
});
