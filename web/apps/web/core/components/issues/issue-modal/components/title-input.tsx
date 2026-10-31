/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormValues } from "../types";
import React from "react";
import { observer } from "mobx-react";
import type { Control, FormState } from "react-hook-form";
import { Controller } from "react-hook-form";
// nerve imports
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
// types
import { useTranslation } from "@nerve/i18n";

// helpers

type TIssueTitleInputProps = {
  control: Control<IssueFormValues>;
  isDraft?: boolean;
  issueTitleRef: React.MutableRefObject<HTMLInputElement | null>;
  formState: FormState<IssueFormValues>;
  handleFormChange: () => void;
};

export const IssueTitleInput = observer(function IssueTitleInput(props: TIssueTitleInputProps) {
  const {
    control,
    isDraft = false,
    issueTitleRef,
    formState: { errors },
    handleFormChange,
  } = props;
  // store hooks
  const { t } = useTranslation();

  const validateWhitespace = (value: string) => {
    if (!isDraft && value.trim() === "") {
      return t("title_is_required");
    }
    if (!isDraft && Array.from(value).length > 255) return t("errors.field.too_long");
    return undefined;
  };
  return (
    <div>
      <Controller
        control={control}
        name="name"
        rules={{
          validate: validateWhitespace,
          required: isDraft ? false : t("title_is_required"),
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
                onChange={(e) => {
                  onChange(e.target.value);
                  handleFormChange();
                }}
                ref={(node) => {
                  ref(node);
                  issueTitleRef.current = node;
                }}
                placeholder={t("title")}
                aria-label={t("title")}
              />
            </InputGroup>
            {errors.name?.message && (
              <FieldError size="2xl" match>
                {errors.name.message}
              </FieldError>
            )}
          </Field>
        )}
      />
    </div>
  );
});
