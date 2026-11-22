/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/** A day of 24 hours, in milliseconds. */
const DAY = 24 * 60 * 60 * 1000;

/** The snoozes a card's menu offers (P4 spec 2.2, R15): for 1, 3, 5, 7 and 14 days, each by its words. */
export const SNOOZE_PRESETS = [
  { days: 1, i18n_label: "notification.snooze.1_day" },
  { days: 3, i18n_label: "notification.snooze.3_days" },
  { days: 5, i18n_label: "notification.snooze.5_days" },
  { days: 7, i18n_label: "notification.snooze.1_week" },
  { days: 14, i18n_label: "notification.snooze.2_weeks" },
] as const;

/**
 * @description the moment a snooze of days chosen at now ends, as nerve takes it (ISO): now and that many days of 24
 * hours, never counted from the day's start nor by the calendar's days, which a change of the clocks lengthens
 */
export const presetMoment = (days: number, now: Date): string => new Date(now.getTime() + days * DAY).toISOString();

/** The half hours of a day: "HH:MM" of 24 hours, 00:00 to 23:30. */
const SLOTS: readonly string[] = Array.from(
  { length: 48 },
  (_, half) => `${String(Math.floor(half / 2)).padStart(2, "0")}:${half % 2 === 0 ? "00" : "30"}`
);

/** The hours and the minutes of a half hour ("HH:MM"). */
const timeOf = (slot: string) => ({ hours: Number(slot.slice(0, 2)), minutes: Number(slot.slice(3, 5)) });

/**
 * @description the moment of the day's half hour slot in the browser's zone, as nerve takes it (ISO): slot is "HH:MM"
 * of 24 hours, so that 00:30 is half past midnight and 12:30 half past noon, never the other way round (W1)
 */
export function slotMoment(day: Date, slot: string): string {
  const { hours, minutes } = timeOf(slot);
  return new Date(day.getFullYear(), day.getMonth(), day.getDate(), hours, minutes).toISOString();
}

/**
 * @description the half hours of the day a snooze may end at, as of now (R15): those whose moment (slotMoment) is
 * after now, so every one of a day after today, today's after now alone, none of a day past; every one while no day
 * is chosen. A half hour the clocks skip that day is none (the start of daylight time: 02:00 and 02:30, whose moments
 * Date moves on to 03:00 and 03:30, whole-branch m4); of the hour the end of daylight time repeats, its first is
 */
export const snoozeSlots = (day: Date | null, now: Date): string[] =>
  day === null
    ? [...SLOTS]
    : SLOTS.filter((slot) => {
        const moment = new Date(slotMoment(day, slot));
        return moment.getHours() === timeOf(slot).hours && moment.getTime() > now.getTime();
      });

/**
 * @description a moment's time of day in the browser's zone as the locale says it (Intl): in 12 hours in English, in 24
 * in Chinese; a half hour of the custom snooze's (slotLabel) and the time a card is snoozed till alike (R15)
 */
export const timeLabel = (moment: Date, locale: string): string =>
  new Intl.DateTimeFormat(locale, { hour: "numeric", minute: "2-digit" }).format(moment);

/** @description a half hour as the locale says it (timeLabel) */
export function slotLabel(slot: string, locale: string): string {
  const { hours, minutes } = timeOf(slot);
  return timeLabel(new Date(2000, 0, 1, hours, minutes), locale);
}
