/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Field, FieldControl, FieldError } from "@makeplane/propel/components/field";
import { useTranslation } from "@nerve/i18n";
import { isComposing } from "@nerve/utils";
import type { TQuickAddIssueForm } from "../root";
export function QuickAddNameField({
  register,
  error,
  className,
}: Pick<TQuickAddIssueForm, "register" | "error"> & { className: string }) {
  const { t } = useTranslation();
  const { ref, ...registration } = register("name", {
    validate: (value) => !!value.trim() || t("issue.title.required"),
  });
  return (
    <div className="w-full">
      <Field name="name" invalid={!!error}>
        <FieldControl
          {...registration}
          type="text"
          autoComplete="off"
          aria-label={t("issue.title.label")}
          placeholder={t("issue.title.label")}
          onKeyDown={(event) => {
            if (event.key === "Enter" && isComposing(event.nativeEvent)) event.preventDefault();
          }}
          render={<input ref={ref} className={className} />}
        />
        {error && (
          <FieldError match size="lg">
            {error}
          </FieldError>
        )}
      </Field>
    </div>
  );
}
