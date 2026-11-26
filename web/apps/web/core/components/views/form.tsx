/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
// nerve imports
import type { View } from "@nerve/api-client";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { ViewsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EmojiIconPickerTypes, EmojiPicker, Logo } from "@nerve/propel/emoji-icon-picker";
import { ModalTitle } from "@nerve/ui";
// components
import { logoPropsOf } from "@/components/project/logo-props";
// lib
import { useRefusedFocus } from "@/lib/refused-focus";

/** What the form holds of a view: who it is, its name, its description (plain text) and its icon. */
export type ViewFormValues = Pick<View, "name" | "description" | "logo_props">;
/** The fields under which nerve's refusals of what the form sent show; it says the others in a toast. */
export type ViewFormField = "name";
/** Where the form says a message of nerve's refusal: under the field (its setError). */
export type ViewUnderField = (field: ViewFormField, message: string) => void;

/** The most characters of a view's name, as nerve counts them: without the white space around it. */
const NAME_LENGTH = 255;

type Props = {
  /** The view a change starts from, as the store holds it as the form opens; none for a new view. */
  view?: View;
  /** Whether the form changes a view (else it makes one). */
  editing: boolean;
  /** Whether what the form sent is out: it is not sent again, nor the form cancelled, until nerve has answered. */
  busy: boolean;
  /** Sends what the form holds, given what it opened with; settles once the page has followed nerve's answer. */
  onSubmit: (values: ViewFormValues, initial: ViewFormValues, underField: ViewUnderField) => Promise<void>;
  onCancel: () => void;
};

/**
 * A view's form (P6a spec 2.4, R28, R30): who the view is and nothing else, its icon, its name and its description;
 * its filters and its display are its page's, which Update view saves. A new view's name, as Save as's, is empty to
 * start with (the caller writes it); a change's is the view's. The name follows nerve's rule: 1 to 255 characters
 * without the white space around it, an emoji one character as nerve counts; it has the focus as the dialog opens. Every control is in the natural Tab order (icon, name, description, Cancel, submit). A name the rules refuse,
 * or one nerve refuses, shows its message under it and takes the focus (useRefusedFocus); a refusal resets nothing.
 */
export function ProjectViewForm(props: Props) {
  const { view, editing, busy, onSubmit, onCancel } = props;
  // nerve hooks
  const { t } = useTranslation();
  // states
  const [pickerOpen, setPickerOpen] = useState(false);
  // what the form opens with, which a change compares what it sends with
  const [initial] = useState<ViewFormValues>(() => ({
    name: view?.name ?? "",
    description: view?.description ?? "",
    logo_props: view?.logo_props ?? {},
  }));
  // form data
  const {
    formState: { errors, submitCount },
    handleSubmit,
    control,
    setError,
  } = useForm<ViewFormValues>({ defaultValues: initial, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const underField: ViewUnderField = (field, message) => setError(field, { type: "server", message });
  const checkName = (value: string) => {
    const name = value.trim();
    if (name === "") return t("title_is_required");
    if (Array.from(name).length > NAME_LENGTH) return t("title_should_be_less_than_255_characters");
    return true;
  };

  return (
    <form
      ref={formRef}
      onSubmit={(event) => void handleSubmit((values) => onSubmit(values, initial, underField))(event)}
    >
      <div className="space-y-5 p-5">
        <ModalTitle className="text-18 font-medium text-secondary">
          {editing ? t("view.update.label") : t("view.create.label")}
        </ModalTitle>
        <div className="space-y-3">
          <div className="flex w-full items-start gap-2">
            <Controller
              control={control}
              name="logo_props"
              render={({ field: { value, onChange } }) => (
                <EmojiPicker
                  ariaLabel={t("aria_labels.view_icon")}
                  iconType="lucide"
                  isOpen={pickerOpen}
                  handleToggle={setPickerOpen}
                  className="flex flex-shrink-0 items-center justify-center"
                  buttonClassName="flex items-center justify-center"
                  label={
                    <span className="grid h-9 w-9 place-items-center rounded-md bg-surface-2">
                      {value.in_use ? (
                        <Logo logo={value} size={18} type="lucide" />
                      ) : (
                        <ViewsOutline className="h-4 w-4 text-tertiary" />
                      )}
                    </span>
                  }
                  onChange={(picked) => {
                    onChange(logoPropsOf(picked));
                    setPickerOpen(false);
                  }}
                  defaultIconColor={value.in_use === "icon" ? value.icon?.color : undefined}
                  defaultOpen={value.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON}
                />
              )}
            />
            <div className="w-full">
              <Controller
                name="name"
                control={control}
                rules={{ validate: checkName }}
                render={({ field: { value, onChange } }) => (
                  <Field name="name" invalid={Boolean(errors.name)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        name="name"
                        type="text"
                        placeholder={t("title")}
                        value={value}
                        onChange={onChange}
                        data-autofocus
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
          </div>
          <Controller
            name="description"
            control={control}
            render={({ field: { value, onChange } }) => (
              <Field name="description">
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    name="description"
                    placeholder={t("description")}
                    aria-label={t("description")}
                    value={value}
                    onChange={onChange}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
        </div>
      </div>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button type="button" variant="secondary" size="lg" onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={busy}>
          {editing
            ? busy
              ? t("common.updating")
              : t("view.update.label")
            : busy
              ? t("common.creating")
              : t("view.create.label")}
        </Button>
      </div>
    </form>
  );
}
