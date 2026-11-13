/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId } from "react";
import type { Control } from "react-hook-form";
import { Controller } from "react-hook-form";
// nerve imports
import { Field, FieldError } from "@makeplane/propel/components/field";
import { StateOutline } from "@makeplane/propel/icons";
import { MODULE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ModuleStatusIcon } from "@nerve/propel/icons";
import { CustomSelect } from "@nerve/ui";
// local imports
import type { ModuleFormValues } from "../form";

type Props = {
  control: Control<ModuleFormValues>;
};

/**
 * A module's status, as one of its form's fields (P6a spec 2.7): a select of the six statuses, required. While its rule
 * fails it says so in a message under it, which describes its select, refused (aria-invalid), so that the form gives it
 * the focus as it gives its other refused fields (useRefusedFocus; M3 closeout handoff 3). Its select is in the natural
 * Tab order.
 */
export function ModuleStatusSelect({ control }: Props) {
  const { t } = useTranslation();
  const messageId = useId();
  return (
    <Controller
      control={control}
      rules={{ required: t("common.errors.required") }}
      name="status"
      render={({ field: { value, onChange }, fieldState: { error } }) => {
        const selected = MODULE_STATUS.find((status) => status.value === value);
        return (
          <Field name="status" invalid={error !== undefined}>
            <CustomSelect
              value={value}
              label={
                <div className="flex items-center justify-center gap-2 py-0.5 text-11">
                  {value ? <ModuleStatusIcon status={value} /> : <StateOutline className="h-3 w-3 text-secondary" />}
                  {selected ? (
                    t(selected.i18n_label)
                  ) : (
                    <span className="text-secondary">{t("project_module.status")}</span>
                  )}
                </div>
              }
              onChange={onChange}
              invalid={error !== undefined}
              describedBy={error?.message === undefined ? undefined : messageId}
              noChevron
            >
              {MODULE_STATUS.map((status) => (
                <CustomSelect.Option key={status.value} value={status.value}>
                  <div className="flex items-center gap-2">
                    <ModuleStatusIcon status={status.value} />
                    {t(status.i18n_label)}
                  </div>
                </CustomSelect.Option>
              ))}
            </CustomSelect>
            {error?.message !== undefined && (
              <FieldError id={messageId} size="lg" match>
                {error.message}
              </FieldError>
            )}
          </Field>
        );
      }}
    />
  );
}
