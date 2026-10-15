/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId } from "react";
import { observer } from "mobx-react";
import { Controller } from "react-hook-form";
import { Field, FieldControl, FieldError } from "@makeplane/propel/components/field";
import { TickCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { ORGANIZATION_SIZE, ORGANIZATION_SIZE_I18N_LABEL } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { Workspace } from "@nerve/api-client";
import { Spinner } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { slugFrom, useCreationForm } from "@/components/workspace/use-create-workspace";
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

  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;

  // form info
  const {
    form: {
      control,
      formState: { errors, isSubmitting, isValid },
    },
    rules,
    onNameChange,
    submit,
  } = useCreationForm((workspace, values) => onCreated(workspace, values.organization_size === "Just myself"));

  const isButtonDisabled = !isValid || isSubmitting;
  // the organization size's label and message, which name and describe its group of buttons
  const organizationSizeLabelId = useId();
  const organizationSizeErrorId = useId();

  if (isWorkspaceCreationDisabled) {
    return (
      <div className="flex flex-col gap-10">
        <span className="text-center text-14 text-tertiary">{t("onboarding.workspace.creation_disabled")}</span>
      </div>
    );
  }
  return (
    <form className="flex flex-col gap-10" onSubmit={(e) => void submit(e)}>
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
            rules={rules.name}
            render={({ field: { value, ref, onChange } }) => (
              // propel's Field, its look kept: the input is Base UI's control (FieldControl) rendered as before, and
              // the message its description (FieldError)
              <Field name="name" invalid={Boolean(errors.name)}>
                <div className="relative flex items-center rounded-md">
                  <FieldControl
                    id="name"
                    name="name"
                    type="text"
                    value={value}
                    onChange={(event) => {
                      onChange(event.target.value);
                      onNameChange(event.target.value);
                    }}
                    placeholder={t("onboarding.workspace.name_placeholder")}
                    // eslint-disable-next-line jsx-a11y/no-autofocus
                    autoFocus
                    render={
                      <input
                        ref={ref}
                        className={cn(
                          "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 placeholder:text-placeholder focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
                          {
                            "border-strong": !errors.name,
                            "border-danger-strong": errors.name,
                          }
                        )}
                      />
                    }
                  />
                </div>
                {errors.name?.message && (
                  <FieldError match size="xl">
                    {errors.name.message}
                  </FieldError>
                )}
              </Field>
            )}
          />
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
            rules={rules.slug}
            render={({ field: { value, ref, onChange } }) => (
              <Field name="slug" invalid={Boolean(errors.slug)}>
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
                  <FieldControl
                    id="slug"
                    name="slug"
                    type="text"
                    value={value}
                    onChange={(e) => onChange(slugFrom(e.target.value))}
                    placeholder={t("workspace_creation.form.url.placeholder")}
                    render={
                      <input
                        ref={ref}
                        className={cn(
                          "ring-none w-full rounded-md border-none bg-surface-1 px-3 py-0 pl-0 text-secondary outline-none placeholder:text-placeholder"
                        )}
                      />
                    }
                  />
                </div>
                <p className="text-13 text-tertiary">{t("workspace_creation.form.url.edit_slug")}</p>
                {errors.slug?.message && (
                  <FieldError match size="xl">
                    {errors.slug.message}
                  </FieldError>
                )}
              </Field>
            )}
          />
        </div>
        <div className="flex flex-col gap-2">
          <label
            id={organizationSizeLabelId}
            className="text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
          >
            {t("workspace_creation.form.organization_size.label")}
          </label>
          <Field name="organization_size" invalid={Boolean(errors.organization_size)}>
            <Controller
              name="organization_size"
              control={control}
              rules={rules.organization_size}
              render={({ field: { value, onChange } }) => (
                // the sizes, a group of buttons (none of Base UI's controls), named by the label and described by the
                // message while one shows
                <div
                  role="group"
                  aria-labelledby={organizationSizeLabelId}
                  aria-describedby={errors.organization_size ? organizationSizeErrorId : undefined}
                  className="flex flex-wrap gap-3"
                >
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

                        <span className="font-medium">{t(ORGANIZATION_SIZE_I18N_LABEL[size])}</span>
                      </button>
                    );
                  })}
                </div>
              )}
            />
            {errors.organization_size?.message && (
              <FieldError id={organizationSizeErrorId} match size="xl">
                {errors.organization_size.message}
              </FieldError>
            )}
          </Field>
        </div>
      </div>
      <Button variant="primary" type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
        {isSubmitting ? <Spinner height="20px" width="20px" /> : t("workspace_creation.button.default")}
      </Button>
    </form>
  );
});
