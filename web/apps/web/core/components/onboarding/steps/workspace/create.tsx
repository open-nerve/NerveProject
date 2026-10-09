/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { TickCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { ORGANIZATION_SIZE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { Workspace } from "@nerve/api-client";
import { Spinner } from "@nerve/ui";
import { cn, validateWorkspaceName } from "@nerve/utils";
// components
import { slugFrom, useCreateWorkspace, type CreationForm } from "@/components/workspace/use-create-workspace";
// hooks
import { useInstance } from "@/hooks/store/use-instance";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  /**
   * What the onboarding does with the workspace created; alone when it is for its creator alone. The step stays busy
   * until it is done, so that no second click checks the new workspace's slug again.
   */
  onCreated: (workspace: Workspace, alone: boolean) => Promise<void>;
};

export const WorkspaceCreateStep = observer(function WorkspaceCreateStep({ onCreated }: Props) {
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { config } = useInstance();
  const create = useCreateWorkspace();

  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;

  // form info
  const {
    handleSubmit,
    control,
    setValue,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<CreationForm>({
    defaultValues: {
      name: "",
      slug: "",
      organization_size: null,
    },
    mode: "onChange",
  });

  const handleCreateWorkspace = async (formData: CreationForm) => {
    const created = await create(formData, setError);
    if (created) await onCreated(created, formData.organization_size === "Just myself");
  };

  const isButtonDisabled = !isValid || isSubmitting;

  if (isWorkspaceCreationDisabled) {
    return (
      <div className="flex flex-col gap-10">
        <span className="text-center text-14 text-tertiary">{t("onboarding.workspace.creation_disabled")}</span>
      </div>
    );
  }
  return (
    <form
      className="flex flex-col gap-10"
      onSubmit={(e) => {
        void handleSubmit(handleCreateWorkspace)(e);
      }}
    >
      <CommonOnboardingHeader
        title={t("workspace_creation.heading")}
        description={t("onboarding.workspace.description")}
      />
      <div className="flex flex-col gap-8">
        <div className="flex flex-col gap-2">
          <label
            className="text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="name"
          >
            {t("workspace_creation.form.name.label")}
          </label>
          <Controller
            control={control}
            name="name"
            rules={{
              required: t("common.errors.required"),
              validate: (value) => validateWorkspaceName(value, true),
              maxLength: {
                value: 80,
                message: t("workspace_creation.errors.validation.name_length"),
              },
            }}
            render={({ field: { value, ref, onChange } }) => (
              <div className="relative flex items-center rounded-md">
                <input
                  id="name"
                  name="name"
                  type="text"
                  value={value}
                  onChange={(event) => {
                    onChange(event.target.value);
                    setValue("name", event.target.value);
                    setValue("slug", slugFrom(event.target.value.trim()), {
                      shouldValidate: true,
                    });
                  }}
                  placeholder={t("onboarding.workspace.name_placeholder")}
                  ref={ref}
                  className={cn(
                    "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 placeholder:text-placeholder focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
                    {
                      "border-strong": !errors.name,
                      "border-danger-strong": errors.name,
                    }
                  )}
                  // eslint-disable-next-line jsx-a11y/no-autofocus
                  autoFocus
                />
              </div>
            )}
          />
          {errors.name && <span className="text-13 text-danger-primary">{errors.name.message}</span>}
        </div>
        <div className="flex flex-col gap-2">
          <label
            className="text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="slug"
          >
            {t("workspace_creation.form.url.label")}
          </label>
          <Controller
            control={control}
            name="slug"
            rules={{
              required: t("common.errors.required"),
              maxLength: {
                value: 48,
                message: t("workspace_creation.errors.validation.url_length"),
              },
            }}
            render={({ field: { value, ref, onChange } }) => (
              <div
                className={cn(
                  "flex w-full items-center rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
                  {
                    "border-strong": !errors.slug,
                    "border-danger-strong": errors.slug,
                  }
                )}
              >
                <span className={cn("rounded-md pr-0 whitespace-nowrap text-secondary")}>
                  {window && window.location.host}/
                </span>
                <input
                  id="slug"
                  name="slug"
                  type="text"
                  value={value}
                  onChange={(e) => onChange(slugFrom(e.target.value))}
                  ref={ref}
                  placeholder={t("workspace_creation.form.url.placeholder")}
                  className={cn(
                    "ring-none w-full rounded-md border-none bg-surface-1 px-3 py-0 pl-0 text-secondary outline-none placeholder:text-placeholder"
                  )}
                />
              </div>
            )}
          />
          <p className="text-13 text-tertiary">{t("workspace_creation.form.url.edit_slug")}</p>
          {errors.slug && <span className="text-13 text-danger-primary">{errors.slug.message}</span>}
        </div>
        <div className="flex flex-col gap-2">
          <label
            className="text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="organization_size"
          >
            {t("workspace_creation.form.organization_size.label")}
          </label>
          <div className="w-full">
            <Controller
              name="organization_size"
              control={control}
              rules={{ required: t("common.errors.required") }}
              render={({ field: { value, onChange } }) => (
                <div className="flex flex-wrap gap-3">
                  {ORGANIZATION_SIZE.map((size) => {
                    const isSelected = value === size;
                    return (
                      <button
                        key={size}
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          onChange(size);
                        }}
                        className={`flex items-center justify-between gap-1 rounded-lg border px-3 py-2 text-13 transition-all duration-200 ${
                          isSelected
                            ? "border-subtle bg-layer-1 text-secondary"
                            : "border-subtle text-tertiary hover:border-strong"
                        }`}
                      >
                        <TickCircleOutline className={cn("size-4 text-placeholder", isSelected && "text-secondary")} />

                        <span className="font-medium">{size}</span>
                      </button>
                    );
                  })}
                </div>
              )}
            />
            {errors.organization_size && (
              <span className="text-13 text-danger-primary">{errors.organization_size.message}</span>
            )}
          </div>
        </div>
      </div>
      <Button variant="primary" type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
        {isSubmitting ? <Spinner height="20px" width="20px" /> : t("workspace_creation.button.default")}
      </Button>
    </form>
  );
});
