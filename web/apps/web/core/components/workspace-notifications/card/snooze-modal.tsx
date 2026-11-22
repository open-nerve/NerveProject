/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useId } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
// nerve imports
import type { Notification } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { CustomSelect, EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { formRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
import { useRefusedFocus } from "@/lib/refused-focus";
// local imports
import { slotLabel, slotMoment, snoozeSlots } from "./snooze-time";
import { useNotificationRefusal } from "./use-notification-refusal";

/** The dialog's fields: a day, and one of its half hours ("HH:MM" of 24 hours). */
type SnoozeForm = { day: Date | null; time: string | null };

/** nerve's field whose refusal the dialog shows under one of its own: the moment, under the time. */
const FIELDS = ["snoozed_till"] as const;

type Props = {
  /** The notification of the card, as nerve last gave it. */
  notification: Notification;
  /** Closes the dialog: the card no longer holds it open. */
  onClose: () => void;
};

/**
 * The custom snooze dialog of a card (P4 spec 2.7, R15), which the card holds open (actions.tsx), never the menu it
 * opens from, so that no timer waits for the menu to close: named by its title (ModalTitle). A day (today at the
 * earliest) and one of its half hours (snoozeSlots: today's after now alone, none the clocks skip, "no time" when none
 * is left), each said in the page's language (slotLabel), 24 hours as nerve is sent them, so that no AM/PM of its own
 * is chosen (W1); the time is named by its field, and once one is picked by its field and the half hour, which it shows
 * (a select's button is named by its content, else by the half hour alone). Either missing says so under it, the
 * control described by it and refused (aria-invalid), the first refused taking the focus (useRefusedFocus). It sends
 * the moment of the day's half hour in the browser's zone (slotMoment) through the store; once nerve has snoozed it,
 * says so and closes; nerve's refusal of the moment (must_be_future) shows under the time, any other in a toast
 * (useNotificationRefusal), the dialog open. It cannot be closed while nerve snoozes (no Escape, no click outside,
 * Cancel out of use). Closed, the focus goes back to what had it before it opened, the row's snooze button
 * (ModalCore). What follows nerve's answer is done only in the session it was sent in (followInSession).
 */
export function NotificationSnoozeModal({ notification, onClose }: Props) {
  const { t, language } = useTranslation();
  const workspace = useRouteWorkspace();
  const { update } = useNotifications();
  const refused = useNotificationRefusal(notification.id);
  const dayMessageId = useId();
  const timeMessageId = useId();
  const {
    control,
    getValues,
    handleSubmit,
    setError,
    setValue,
    formState: { isSubmitting, submitCount },
  } = useForm<SnoozeForm>({ defaultValues: { day: null, time: null }, shouldFocusError: false });
  // the first field refused takes the focus once its message shows, after the rules' check and after nerve's refusal
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const now = new Date();
  const chosenDay = useWatch({ control, name: "day" });
  const slots = snoozeSlots(chosenDay, now);
  const required = { required: t("common.errors.required") };

  const onSubmit = async ({ day, time }: SnoozeForm) => {
    if (!workspace || day === null || time === null) return;
    await followInSession(() => update(workspace, notification.id, { snoozed_till: slotMoment(day, time) }), {
      done: () => {
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("notification.toasts.snoozed") });
        onClose();
      },
      failed: (error) => {
        const refusal = formRefusal(error, FIELDS, {});
        const message = refusal.kind === "fields" ? refusal.fields.snoozed_till : undefined;
        if (message === undefined) refused(error);
        else setError("time", { type: "server", message: t(message) });
      },
    });
  };

  return (
    <ModalCore
      isOpen
      handleClose={isSubmitting ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <form ref={formRef} className="p-5" onSubmit={(event) => void handleSubmit(onSubmit)(event)}>
        <ModalTitle className="text-h5-medium text-primary">{t("notification.snooze_modal.title")}</ModalTitle>
        <div className="mt-5 flex flex-col gap-3 md:flex-row md:items-start">
          <Controller
            control={control}
            name="day"
            rules={required}
            render={({ field: { value, onChange }, fieldState: { error } }) => (
              <div className="flex-1">
                <Field name="day" invalid={error !== undefined}>
                  <DateDropdown
                    value={value}
                    placeholder={t("notification.snooze_modal.date")}
                    onChange={(day) => {
                      onChange(day);
                      // a half hour the day no longer offers (today's, gone by) is chosen no more
                      const time = getValues("time");
                      if (time !== null && !snoozeSlots(day, new Date()).includes(time)) setValue("time", null);
                    }}
                    minDate={new Date(now.getFullYear(), now.getMonth(), now.getDate())}
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
          <Controller
            control={control}
            name="time"
            rules={required}
            render={({ field: { value, onChange }, fieldState: { error } }) => (
              <div className="flex-1">
                <Field name="time" invalid={error !== undefined}>
                  <CustomSelect
                    value={value}
                    onChange={onChange}
                    label={
                      value === null ? (
                        <span className="text-placeholder">{t("notification.snooze_modal.time")}</span>
                      ) : (
                        // named by its field and the half hour picked; seen as the half hour alone
                        <>
                          <span className="sr-only">
                            {t("notification.snooze_modal.picked_time", { time: slotLabel(value, language) })}
                          </span>
                          <span aria-hidden="true">{slotLabel(value, language)}</span>
                        </>
                      )
                    }
                    input
                    invalid={error !== undefined}
                    describedBy={error?.message === undefined ? undefined : timeMessageId}
                  >
                    {slots.length > 0 ? (
                      slots.map((slot) => (
                        <CustomSelect.Option key={slot} value={slot}>
                          {slotLabel(slot, language)}
                        </CustomSelect.Option>
                      ))
                    ) : (
                      <p className="p-3 text-center text-secondary">{t("notification.snooze_modal.no_time")}</p>
                    )}
                  </CustomSelect>
                  {error?.message !== undefined && (
                    <FieldError id={timeMessageId} size="lg" match>
                      {error.message}
                    </FieldError>
                  )}
                </Field>
              </div>
            )}
          />
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" size="lg" disabled={isSubmitting} onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {t("notification.snooze_modal.submit")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
