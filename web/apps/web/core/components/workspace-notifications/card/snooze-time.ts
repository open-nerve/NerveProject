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
