/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useId } from "react";
import { Controller, useWatch, type Control } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { useTranslation } from "@nerve/i18n";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
import { DateDropdown } from "@/components/dropdowns/date";
import type { IssueFormValues } from "../types";

/** The paired date fields validate the pair while retaining the original single-field editing controls. */
export function IssueDateFields({
  control,
  isDraft,
  handleFormChange,
}: {
  control: Control<IssueFormValues>;
  isDraft: boolean;
  handleFormChange: () => void;
}) {
  const id = useId();
  const { t } = useTranslation();
  const [start, target] = useWatch({ control, name: ["start_date", "target_date"] });
  return (["start_date", "target_date"] as const).map((name) => {
    const messageId = `${id}-${name}`;
    return (
      <Controller
        key={name}
        name={name}
        control={control}
        rules={{
          deps: name === "start_date" ? "target_date" : "start_date",
          validate: (value) => {
            const from = name === "start_date" ? value : start;
            const to = name === "target_date" ? value : target;
            return isDraft || !from || !to || from <= to || t("issue.calendar.due_before_start");
          },
        }}
        render={({ field: { value, onChange, onBlur }, fieldState: { error } }) => (
          <Field name={name} invalid={!!error}>
            <div className="h-7">
              <DateDropdown
                value={value}
                onChange={(date) => {
                  onChange(date ? renderFormattedPayloadDate(date) : null);
                  handleFormChange();
                }}
                onClose={onBlur}
                buttonVariant="border-with-text"
                minDate={name === "target_date" ? (getDate(start) ?? undefined) : undefined}
                maxDate={name === "start_date" ? (getDate(target) ?? undefined) : undefined}
                placeholder={t(name === "start_date" ? "start_date" : "due_date")}
                invalid={!!error}
                describedBy={error ? messageId : undefined}
              />
            </div>
            {error?.message && (
              <FieldError id={messageId} size="lg" match>
                {error.message}
              </FieldError>
            )}
          </Field>
        )}
      />
    );
  });
}
