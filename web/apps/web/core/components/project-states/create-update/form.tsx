/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { StateCreate } from "@nerve/api-client";
import { Popover } from "@nerve/ui";
// components
import { ColorPicker } from "@/components/common/color-picker";
// lib
import { useRefusedFocus } from "@/lib/refused-focus";
// local imports
import type { StateUnderField } from "../use-state-changes";

/** What the form edits of a state, as a creation sends it. */
export type TStateFormData = Omit<StateCreate, "group">;

type TStateForm = {
  data: TStateFormData;
  /**
   * Sends what the form holds; settles once the page has followed nerve's answer, the form busy until then. nerve's
   * refusal shows under the fields it names, by underField.
   */
  onSubmit: (formData: TStateFormData, underField: StateUnderField) => Promise<void>;
  onCancel: () => void;
  /** What the form's button says, and says while the form is busy. */
  buttonTitle: string;
  busyButtonTitle: string;
};

function PopoverButton({ color }: { color?: string }) {
  return (
    <div
      className="group inline-flex h-5 w-5 items-center rounded-sm text-14 font-medium transition-all focus:outline-none"
      style={{
        backgroundColor: color ?? "black",
      }}
    />
  );
}

export function StateForm(props: TStateForm) {
  const { data, onSubmit, onCancel, buttonTitle, busyButtonTitle } = props;
  const { t } = useTranslation();
  const {
    control,
    handleSubmit,
    setError,
    watch,
    setFocus,
    formState: { errors, isSubmitting, submitCount },
  } = useForm<TStateFormData>({ defaultValues: data, shouldFocusError: false });
  // the first field refused takes the focus once its message shows (useRefusedFocus), not react-hook-form's before
  const formRef = useRefusedFocus<HTMLDivElement>(submitCount);

  // the name has the focus once the form shows
  useEffect(() => {
    setFocus("name");
  }, [setFocus]);

  // busy from the click until the page has followed nerve's answer: a second click finds the button disabled
  const submit = handleSubmit((formData) =>
    onSubmit(formData, (field, message) => setError(field, { type: "server", message }))
  );

  return (
    <div ref={formRef} className="relative flex space-x-2 rounded-sm bg-surface-1 p-3">
      {/* color */}
      <div className="mt-2 h-full flex-shrink-0">
        <Popover
          button={<PopoverButton color={watch("color")} />}
          buttonAriaLabel={t("common.color")}
          panelClassName="mt-4 -ml-3"
        >
          <Controller
            name="color"
            control={control}
            render={({ field: { value, onChange } }) => (
              <ColorPicker color={value} onChange={(picked) => onChange(picked.hex)} />
            )}
          />
        </Popover>
      </div>

      <div className="w-full space-y-2">
        {/* title */}
        <Controller
          name="name"
          control={control}
          rules={{ required: t("errors.field.required") }}
          render={({ field: { value, onChange, ref } }) => (
            <Field name="name" invalid={Boolean(errors.name)}>
              <InputGroup size="2xl">
                <Input
                  size="2xl"
                  id="name"
                  type="text"
                  name="name"
                  placeholder={t("name")}
                  value={value}
                  onChange={onChange}
                  ref={ref}
                  maxLength={100}
                />
              </InputGroup>
              {errors.name?.message && (
                <FieldError match size="lg">
                  {errors.name.message}
                </FieldError>
              )}
            </Field>
          )}
        />

        {/* description */}
        <Controller
          name="description"
          control={control}
          render={({ field: { value, onChange, ref } }) => (
            <Field name="description" invalid={Boolean(errors.description)}>
              <TextAreaGroup resize="none">
                <TextArea
                  ref={ref}
                  size="lg"
                  surface="field"
                  autoResize
                  maxRows={8}
                  id="description"
                  name="description"
                  placeholder={t("project_settings.states.description_placeholder")}
                  value={value ?? ""}
                  onChange={onChange}
                />
              </TextAreaGroup>
              {errors.description?.message && (
                <FieldError match size="lg">
                  {errors.description.message}
                </FieldError>
              )}
            </Field>
          )}
        />

        <div className="flex items-center space-x-2">
          <Button onClick={submit} variant="primary" size="lg" loading={isSubmitting}>
            {isSubmitting ? busyButtonTitle : buttonTitle}
          </Button>
          <Button type="button" variant="secondary" size="lg" disabled={isSubmitting} onClick={onCancel}>
            {t("cancel")}
          </Button>
        </div>
      </div>
    </div>
  );
}
