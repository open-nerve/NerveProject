/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller } from "react-hook-form";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { ORGANIZATION_SIZE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { Workspace } from "@nerve/api-client";
// ui
import { CustomSelect } from "@nerve/ui";
import { useNavigate } from "react-router";
// local imports
import { slugFrom, useCreationForm } from "./use-create-workspace";

type Props = {
  /**
   * What the page does with the workspace created (in the session it was created in): the form stays busy until it
   * is done, so that no second click checks the new workspace's slug again.
   */
  onCreated: (workspace: Workspace) => Promise<void>;
};

export const CreateWorkspaceForm = observer(function CreateWorkspaceForm(props: Props) {
  const { t } = useTranslation();
  const { onCreated } = props;
  // router
  const navigate = useNavigate();
  // form info
  const {
    form: {
      control,
      formState: { errors, isSubmitting, isValid },
    },
    rules,
    onNameChange,
    submit,
  } = useCreationForm(onCreated);

  return (
    <form className="space-y-6 sm:space-y-9" onSubmit={(e) => void submit(e)}>
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
              rules={rules.name}
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
                        onNameChange(e.target.value);
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
            rules={rules.slug}
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
              rules={rules.organization_size}
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
