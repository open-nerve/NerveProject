/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// lib
import { formRefusal } from "@/lib/form-refusal";
import { useRefusedFocus } from "@/lib/refused-focus";
// store
import { snoozeDay, snoozeInstant } from "@/store/issue/intake/status";

/** nerve's field whose refusal the modal shows under its own: the moment, under the day. */
const FIELDS = ["snoozed_till"] as const;

/** The modal's form: the day chosen, which the modal sends the first moment of. */
type SnoozeForm = { day: Date };

type Props = {
  /** The snooze the work item holds, if it is snoozed: the modal starts at its day (snoozeDay). */
  value: string | null;
  /** Closes the modal: a cancel. */
  onClose: () => void;
  /**
   * Snoozes the work item till that moment (the action model's triage), settling once nerve has (the modal is then
   * closed by it) and rejecting on a refusal, which it says unless underField shows it under the day.
   */
  onSubmit: (snoozedTill: string, underField: (error: unknown) => boolean) => Promise<void>;
};

/**
 * The snooze modal of a work item of a project's intake (P5a spec 2.8, R13), named by its title (ModalTitle): a day of
 * the caller's calendar, tomorrow at the earliest (nerve answers must_be_future for a moment not after the request),
 * at first tomorrow, or the day of the snooze held while that is tomorrow or later (snoozeDay); it sends that day's
 * first moment in the browser's zone (snoozeInstant). nerve's refusal of the moment shows under the day, which is then
 * refused (aria-invalid) and takes the focus (useRefusedFocus); any other refusal is the action model's to say, the
 * modal open. It cannot be closed while nerve snoozes (no Escape, no click outside, Cancel out of use).
 */
export function InboxIssueSnoozeModal({ value, onClose, onSubmit }: Props) {
  const { t } = useTranslation();
  const dayMessageId = useId();
  // the earliest day, as the modal opened: tomorrow
  const [earliest] = useState(() => snoozeDay(new Date()));
  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting, submitCount },
  } = useForm<SnoozeForm>({ defaultValues: { day: snoozeDay(new Date(), value) }, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);

  /** Shows nerve's refusal of the moment under the day; whether it did. */
  const underDay = (error: unknown): boolean => {
    const refusal = formRefusal(error, FIELDS, {});
    const message = refusal.kind === "fields" ? refusal.fields.snoozed_till : undefined;
    if (message === undefined) return false;
    setError("day", { type: "server", message: t(message) });
    return true;
  };
  const snooze = async ({ day }: SnoozeForm) => {
    // a refusal is said by the action model, or under the day
    await onSubmit(snoozeInstant(day), underDay).catch(() => undefined);
  };

  return (
    <ModalCore
      isOpen
      handleClose={isSubmitting ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.SM}
    >
      <form ref={formRef} className="p-5" onSubmit={(event) => void handleSubmit(snooze)(event)}>
        <ModalTitle className="text-h5-medium text-primary">{t("inbox_issue.modals.snooze.title")}</ModalTitle>
        <Controller
          control={control}
          name="day"
          render={({ field: { value: day, onChange }, fieldState: { error } }) => (
            <div className="mt-5">
              <Field name="day" invalid={error !== undefined}>
                <DateDropdown
                  value={day}
                  placeholder={t("inbox_issue.modals.snooze.day")}
                  onChange={(chosen) => {
                    if (chosen) onChange(chosen);
                  }}
                  minDate={earliest}
                  isClearable={false}
                  buttonVariant="border-with-text"
                  buttonContainerClassName="w-full text-left"
                  buttonClassName="border-strong px-3 py-2"
                  invalid={error !== undefined}
                  describedBy={error?.message === undefined ? undefined : dayMessageId}
                />
                {error?.message !== undefined && (
                  <FieldError id={dayMessageId} size="lg" match>
                    {error.message}
                  </FieldError>
                )}
              </Field>
            </div>
          )}
        />
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" size="lg" disabled={isSubmitting} onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {t("inbox_issue.actions.snooze")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
