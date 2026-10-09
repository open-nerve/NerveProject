/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, SetStateAction } from "react";
import { useEffect } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { ORGANIZATION_SIZE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { Workspace } from "@nerve/api-client";
// ui
import { CustomSelect } from "@nerve/ui";
import { validateWorkspaceName } from "@nerve/utils";
import { useNavigate } from "react-router";
// local imports
import { slugFrom, useCreateWorkspace, type CreationForm } from "./use-create-workspace";

type Props = {
  /**
   * What the page does with the workspace created (in the session it was created in): the form stays busy until it
   * is done, so that no second click checks the new workspace's slug again.
   */
  onCreated: (workspace: Workspace) => Promise<void>;
  defaultValues: CreationForm;
  setDefaultValues: Dispatch<SetStateAction<CreationForm>>;
};

export const CreateWorkspaceForm = observer(function CreateWorkspaceForm(props: Props) {
  const { t } = useTranslation();
  const { onCreated, defaultValues, setDefaultValues } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const create = useCreateWorkspace();
  // form info
  const {
    handleSubmit,
    control,
    setValue,
    getValues,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<CreationForm>({ defaultValues, mode: "onChange" });

  const handleCreateWorkspace = async (formData: CreationForm) => {
    const created = await create(formData, setError);
    if (created) await onCreated(created);
  };

  useEffect(
    () => () => {
      // when the component unmounts set the default values to whatever user typed in
      setDefaultValues(getValues());
    },
    [getValues, setDefaultValues]
  );

  return (
    <form
      className="space-y-6 sm:space-y-9"
      onSubmit={(e) => {
        void handleSubmit(handleCreateWorkspace)(e);
      }}
    >
      <div className="space-y-6 sm:space-y-7">
        <div className="flex flex-col gap-2 text-13">
          <label htmlFor="workspaceName">
            {t("workspace_creation.form.name.label")}
            <span className="ml-0.5 text-danger-primary">*</span>
          </label>
          <div className="flex flex-col gap-1">
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
                <Field name="workspaceName" invalid={Boolean(errors.name)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      id="workspaceName"
                      type="text"
                      value={value}
                      onChange={(e) => {
                        onChange(e.target.value);
                        setValue("name", e.target.value);
                        setValue("slug", slugFrom(e.target.value.trim()), {
                          shouldValidate: true,
                        });
                      }}
                      ref={ref}
                      placeholder={t("workspace_creation.form.name.placeholder")}
                    />
                  </InputGroup>
                </Field>
              )}
            />
            <span className="text-11 text-danger-primary">{errors?.name?.message}</span>
          </div>
        </div>
        <div className="flex flex-col gap-2 text-13">
          <label htmlFor="workspaceUrl">
            {t("workspace_creation.form.url.label")}
            <span className="ml-0.5 text-danger-primary">*</span>
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
            render={({ field: { onChange, value, ref } }) => (
              <Field name="workspaceUrl" invalid={Boolean(errors.slug)}>
                <InputGroup size="2xl">
                  <span className="text-12 whitespace-nowrap text-secondary">{window && window.location.host}/</span>
                  <Input
                    size="2xl"
                    id="workspaceUrl"
                    type="text"
                    value={value}
                    onChange={(e) => onChange(slugFrom(e.target.value))}
                    ref={ref}
                    placeholder={t("workspace_creation.form.url.placeholder")}
                  />
                </InputGroup>
              </Field>
            )}
          />
          {errors.slug && <span className="text-11 text-danger-primary">{errors.slug.message}</span>}
        </div>
        <div className="flex flex-col gap-2 text-13">
          <span>
            {t("workspace_creation.form.organization_size.label")}
            <span className="ml-0.5 text-danger-primary">*</span>
          </span>
          <div className="w-full">
            <Controller
              name="organization_size"
              control={control}
              rules={{ required: t("common.errors.required") }}
              render={({ field: { value, onChange } }) => (
                <CustomSelect
                  value={value}
                  onChange={onChange}
                  label={
                    ORGANIZATION_SIZE.find((c) => c === value) ?? (
                      <span className="text-placeholder">
                        {t("workspace_creation.form.organization_size.placeholder")}
                      </span>
                    )
                  }
                  buttonClassName="border border-subtle bg-layer-2 !shadow-none !rounded-md"
                  input
                >
                  {ORGANIZATION_SIZE.map((item) => (
                    <CustomSelect.Option key={item} value={item}>
                      {item}
                    </CustomSelect.Option>
                  ))}
                </CustomSelect>
              )}
            />
            {errors.organization_size && (
              <span className="text-13 text-danger-primary">{errors.organization_size.message}</span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <Button variant="primary" type="submit" size="xl" disabled={!isValid} loading={isSubmitting}>
          {isSubmitting ? t("workspace_creation.button.loading") : t("workspace_creation.button.default")}
        </Button>
        <Button variant="secondary" type="button" size="xl" onClick={() => navigate(-1)}>
          {t("common.go_back")}
        </Button>
      </div>
    </form>
  );
});
