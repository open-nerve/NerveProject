/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { Control, FieldArrayWithId, FormState } from "react-hook-form";
import { Controller } from "react-hook-form";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { WorkspaceInvitationsCreate } from "@nerve/api-client";
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CloseOutline } from "@makeplane/propel/icons";
import { CustomSelect } from "@nerve/ui";
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";

type TInvitationFieldsProps = {
  fields: FieldArrayWithId<WorkspaceInvitationsCreate, "invitations", "id">[];
  control: Control<WorkspaceInvitationsCreate>;
  formState: FormState<WorkspaceInvitationsCreate>;
  remove: (index: number) => void;
};

export const InvitationFields = observer(function InvitationFields(props: TInvitationFieldsProps) {
  const {
    fields,
    control,
    formState: { errors },
    remove,
  } = props;
  // nerve hooks
  const { t } = useTranslation();

  return (
    <div className="mb-3 space-y-4">
      {fields.map((field, index) => (
        <div
          key={field.id}
          className="group relative mb-1 flex w-full items-start justify-between gap-x-4 text-body-xs-regular"
        >
          <div className="w-full">
            <Controller
              control={control}
              name={`invitations.${index}.email`}
              rules={{
                required: t("workspace_settings.settings.members.modal.errors.required"),
                pattern: {
                  value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                  message: t("workspace_settings.settings.members.modal.errors.invalid"),
                },
              }}
              render={({ field: { value, onChange, ref } }) => (
                <>
                  <Field name="input" invalid={Boolean(errors.invitations?.[index]?.email)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id={`invitations.${index}.email`}
                        name={`invitations.${index}.email`}
                        type="text"
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder={t("workspace_settings.settings.members.modal.placeholder")}
                      />
                    </InputGroup>
                  </Field>
                  {errors.invitations?.[index]?.email && (
                    <span className="ml-1 text-caption-sm-regular text-danger-primary">
                      {errors.invitations?.[index]?.email?.message}
                    </span>
                  )}
                </>
              )}
            />
          </div>
          <div className="flex shrink-0 items-center justify-between gap-2">
            <div className="flex flex-col gap-1">
              <Controller
                control={control}
                name={`invitations.${index}.role`}
                rules={{ required: true }}
                render={({ field: { value, onChange } }) => (
                  <CustomSelect
                    value={value}
                    label={
                      <span className="text-caption-sm-regular sm:text-body-xs-regular">
                        {t(ROLE_DETAILS[value].i18n_title)}
                      </span>
                    }
                    onChange={onChange}
                    className="w-24 flex-grow"
                    input
                  >
                    {/* every role: only an admin invites (the members page's gate), and an admin may give any; the select
                    gives the picked option's value, a role's number */}
                    {WORKSPACE_ROLES.map((role) => (
                      <CustomSelect.Option key={role} value={role}>
                        {t(ROLE_DETAILS[role].i18n_title)}
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                )}
              />
            </div>
            {fields.length > 1 && (
              <div className="flex-item flex w-6">
                <button
                  type="button"
                  className="place-items-center self-center rounded-sm"
                  onClick={() => remove(index)}
                >
                  <CloseOutline className="h-4 w-4 text-secondary" />
                </button>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
});
