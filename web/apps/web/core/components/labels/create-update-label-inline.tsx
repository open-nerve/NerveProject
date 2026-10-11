/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { forwardRef, useEffect } from "react";
import { observer } from "mobx-react";
import { TwitterPicker } from "react-color";
import { Controller, useForm } from "react-hook-form";
import { Popover, Transition } from "@headlessui/react";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { getRandomLabelColor, LABEL_COLOR_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { Label } from "@nerve/api-client";
// local imports
import { useLabelChanges, type LabelUnderField } from "./use-label-changes";

/** What the form edits and sends, a creation (at the top) or a change alike: nerve's LabelUpdate takes no others. */
type TLabelFormValues = Pick<Label, "name" | "color">;

type TCreateUpdateLabelInlineProps = {
  /** The project a label created goes in. */
  projectId: string;
  labelForm: boolean;
  setLabelForm: React.Dispatch<React.SetStateAction<boolean>>;
  /** The label the form changes; none, a creation. */
  labelToUpdate?: Label;
  onClose?: () => void;
};

const defaultValues: TLabelFormValues = {
  name: "",
  color: "var(--text-color-secondary)",
};

export const CreateUpdateLabelInline = observer(
  forwardRef(function CreateUpdateLabelInline(
    props: TCreateUpdateLabelInlineProps,
    ref: React.ForwardedRef<HTMLDivElement>
  ) {
    const { projectId, labelForm, setLabelForm, labelToUpdate, onClose } = props;
    // form info
    const {
      handleSubmit,
      control,
      reset,
      setError,
      formState: { errors, isSubmitting },
      watch,
      setFocus,
    } = useForm<TLabelFormValues>({
      // a change starts from the label's name and colour as nerve holds them (none, for a label made without one), a
      // creation from no name and a colour at random
      defaultValues: labelToUpdate
        ? { name: labelToUpdate.name, color: labelToUpdate.color }
        : { ...defaultValues, color: getRandomLabelColor() },
    });

    const { t } = useTranslation();
    const { create, update } = useLabelChanges();

    const handleClose = () => {
      setLabelForm(false);
      reset(defaultValues);
      if (onClose) onClose();
    };

    // Busy until the page has followed nerve's answer: the form closes once nerve has made the label or the change,
    // and stays with what it holds when nerve refuses it, the reason under the name or in a toast.
    const handleFormSubmit = (formData: TLabelFormValues) => {
      const underField: LabelUnderField = (field, message) => setError(field, { type: "server", message });
      return labelToUpdate
        ? update(labelToUpdate.id, formData, underField, handleClose)
        : create(projectId, formData, underField, handleClose);
    };

    /**
     * For settings focus on name input
     */
    useEffect(() => {
      setFocus("name");
    }, [setFocus, labelForm]);

    return (
      <>
        <div
          ref={ref}
          className={`flex w-full scroll-m-8 items-center gap-2 bg-surface-1 ${labelForm ? "" : "hidden"}`}
        >
          <div className="flex-shrink-0">
            <Popover className="relative z-10 flex h-full w-full items-center justify-center">
              {({ open }) => (
                <>
                  <Popover.Button
                    aria-label={t("common.color")}
                    className={`group inline-flex items-center text-14 font-medium ${
                      open ? "text-primary" : "text-secondary"
                    }`}
                  >
                    {/* a label without a colour shown as the list shows it (label-name.tsx) */}
                    <span
                      className="h-4 w-4 rounded-full"
                      style={{
                        backgroundColor: watch("color") || "#000",
                      }}
                    />
                  </Popover.Button>

                  <Transition
                    as={React.Fragment}
                    enter="transition ease-out duration-200"
                    enterFrom="opacity-0 translate-y-1"
                    enterTo="opacity-100 translate-y-0"
                    leave="transition ease-in duration-150"
                    leaveFrom="opacity-100 translate-y-0"
                    leaveTo="opacity-0 translate-y-1"
                  >
                    <Popover.Panel className="absolute top-full left-0 z-20 mt-3 w-screen max-w-xs px-2 sm:px-0">
                      <Controller
                        name="color"
                        control={control}
                        render={({ field: { value, onChange } }) => (
                          <TwitterPicker
                            colors={LABEL_COLOR_OPTIONS}
                            color={value}
                            onChange={(picked) => onChange(picked.hex)}
                          />
                        )}
                      />
                    </Popover.Panel>
                  </Transition>
                </>
              )}
            </Popover>
          </div>
          <div className="flex flex-1 flex-col justify-center">
            <Controller
              control={control}
              name="name"
              rules={{
                required: t("project_settings.labels.label_title_is_required"),
                maxLength: {
                  value: 255,
                  message: t("project_settings.labels.label_max_char"),
                },
              }}
              render={({ field: { value, onChange, ref: inputRef } }) => (
                <Field name="name" invalid={Boolean(errors.name)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      id="labelName"
                      name="name"
                      type="text"
                      value={value}
                      onChange={onChange}
                      ref={inputRef}
                      placeholder={t("project_settings.labels.label_title")}
                    />
                  </InputGroup>
                </Field>
              )}
            />
          </div>
          <Button variant="secondary" onClick={() => handleClose()} disabled={isSubmitting}>
            {t("cancel")}
          </Button>
          <Button
            variant="primary"
            onClick={(e) => {
              e.preventDefault();
              void handleSubmit(handleFormSubmit)();
            }}
            loading={isSubmitting}
          >
            {labelToUpdate ? (isSubmitting ? t("updating") : t("update")) : isSubmitting ? t("adding") : t("add")}
          </Button>
        </div>
        {errors.name?.message && <p className="p-0.5 pl-8 text-13 text-danger-primary">{errors.name?.message}</p>}
      </>
    );
  })
);
