/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { add } from "date-fns";
import { Controller, useForm } from "react-hook-form";
import { CalendarOutline } from "@makeplane/propel/icons";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import type { ApiTokenCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// ui
import { Switch } from "@makeplane/propel/components/switch";
import { CustomSelect } from "@nerve/ui";
import { cn, renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// helpers
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
// local imports
import type { TExpiryChoice } from "./expiry";
import { EXPIRY_PERIODS, expiryDate } from "./expiry";

type Props = {
  handleClose: () => void;
  neverExpires: boolean;
  toggleNeverExpires: () => void;
  /** Creates the token; fails as nerve refuses. */
  onSubmit: (data: ApiTokenCreate) => Promise<void>;
};

type TFormValues = {
  label: string;
  description: string;
  expiry: TExpiryChoice | null;
};

const defaultValues: TFormValues = {
  label: "",
  description: "",
  expiry: null,
};

/** The fields of ApiTokenCreate whose errors show under the form's fields. */
const FIELDS = ["label", "expired_at"] as const;

export function CreateApiTokenForm(props: Props) {
  const { handleClose, neverExpires, toggleNeverExpires, onSubmit } = props;
  // states
  const [customDate, setCustomDate] = useState<Date | null>(null);
  // form
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    setError,
    watch,
  } = useForm<TFormValues>({ defaultValues });
  // hooks
  const { t } = useTranslation();

  const handleFormSubmit = async (data: TFormValues) => {
    const expiresAt = data.expiry === null ? undefined : expiryDate(data.expiry, new Date(), customDate);
    if (!neverExpires && expiresAt === undefined) {
      setError("expiry", { type: "manual", message: t("account_settings.api_tokens.expiry.required") });
      return;
    }
    try {
      await onSubmit({
        label: data.label,
        description: data.description,
        expired_at: neverExpires ? null : expiresAt?.toISOString(),
      });
    } catch (error) {
      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
      const fields = fieldErrorKeys(error);
      if (fields.label !== undefined) setError("label", { type: "manual", message: t(fields.label) });
      if (fields.expired_at !== undefined) setError("expiry", { type: "manual", message: t(fields.expired_at) });
      if (needsErrorBanner(error, FIELDS))
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    }
  };

  const tomorrow = add(new Date(), { days: 1 });
  const expiry = watch("expiry");
  const expiresAt = expiry === null ? undefined : expiryDate(expiry, new Date(), customDate);

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)}>
      <div className="space-y-5 p-5">
        <h3 className="text-18 font-medium text-secondary">
          {t("workspace_settings.settings.api_tokens.create_token")}
        </h3>
        <div className="space-y-3">
          <div className="space-y-1">
            <Controller
              control={control}
              name="label"
              rules={{
                required: t("title_is_required"),
                maxLength: {
                  value: 255,
                  message: t("title_should_be_less_than_255_characters"),
                },
                validate: (val) => val.trim() !== "" || t("title_is_required"),
              }}
              render={({ field: { value, onChange } }) => (
                <Field name="input" invalid={Boolean(errors.label)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      type="text"
                      value={value}
                      onChange={onChange}
                      placeholder={t("title")}
                      aria-label={t("title")}
                    />
                  </InputGroup>
                </Field>
              )}
            />
            {errors.label && <span className="text-11 text-danger-primary">{errors.label.message}</span>}
          </div>
          <Controller
            control={control}
            name="description"
            render={({ field: { value, onChange } }) => (
              <Field name="description" invalid={Boolean(errors.description)}>
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    value={value}
                    onChange={onChange}
                    placeholder={t("description")}
                    aria-label={t("description")}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
          <div className="space-y-1">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Controller
                  control={control}
                  name="expiry"
                  render={({ field: { onChange, value } }) => (
                    <CustomSelect
                      customButton={
                        <div
                          className={cn(
                            "flex h-7 items-center gap-2 rounded-sm border-[0.5px] border-strong px-2 py-0.5",
                            {
                              "text-placeholder": neverExpires,
                            }
                          )}
                        >
                          <CalendarOutline className="h-3 w-3" />
                          {t(`account_settings.api_tokens.expiry.${value ?? "set"}`)}
                        </div>
                      }
                      value={value}
                      onChange={onChange}
                      disabled={neverExpires}
                    >
                      {EXPIRY_PERIODS.map((option) => (
                        <CustomSelect.Option key={option.key} value={option.key}>
                          {t(`account_settings.api_tokens.expiry.${option.key}`)}
                        </CustomSelect.Option>
                      ))}
                      <CustomSelect.Option value="custom">
                        {t("account_settings.api_tokens.expiry.custom")}
                      </CustomSelect.Option>
                    </CustomSelect>
                  )}
                />
                {expiry === "custom" && (
                  <div className="h-7">
                    <DateDropdown
                      value={customDate}
                      onChange={(date) => setCustomDate(date)}
                      minDate={tomorrow}
                      icon={<CalendarOutline className="h-3 w-3" />}
                      buttonVariant="border-with-text"
                      placeholder={t("account_settings.api_tokens.expiry.set_date")}
                      disabled={neverExpires}
                    />
                  </div>
                )}
              </div>
              {!neverExpires && expiresAt && (
                <span className="text-11 text-placeholder">
                  {t("account_settings.api_tokens.expires_at", {
                    date: renderFormattedDate(expiresAt),
                    time: renderFormattedTime(expiresAt),
                  })}
                </span>
              )}
            </div>
            {!neverExpires && errors.expiry && (
              <span className="text-11 text-danger-primary">{errors.expiry.message}</span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <label className="flex cursor-pointer items-center gap-1.5">
          <Switch size="sm" checked={neverExpires} onCheckedChange={toggleNeverExpires} />
          <span className="text-11">{t("workspace_settings.settings.api_tokens.never_expires")}</span>
        </label>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={handleClose}>
            {t("cancel")}
          </Button>
          <Button variant="primary" type="submit" loading={isSubmitting}>
            {isSubmitting
              ? t("workspace_settings.settings.api_tokens.generating")
              : t("workspace_settings.settings.api_tokens.generate_token")}
          </Button>
        </div>
      </div>
    </form>
  );
}
