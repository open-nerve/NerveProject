/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { add, set } from "date-fns";

/** The periods a new token may last (M2 design 7.7); it may also end on a day picked, or never. */
export const EXPIRY_PERIODS = [
  { key: "1_week", period: { weeks: 1 } },
  { key: "1_month", period: { months: 1 } },
  { key: "3_months", period: { months: 3 } },
  { key: "1_year", period: { years: 1 } },
] as const;

/** A period's key, or "custom" for a day picked. */
export type TExpiryChoice = (typeof EXPIRY_PERIODS)[number]["key"] | "custom";

/**
 * When a token created at now expires: the period chosen after now, or the day picked (a date at midnight, as
 * the date picker gives it) at now's time of day. Undefined for "custom" before a day is picked. The time of day
 * is set on the day, not added to its midnight: on the day summer time starts or ends, the hours since midnight
 * are one more or one fewer than the clock shows.
 */
export function expiryDate(choice: TExpiryChoice, now: Date, picked: Date | null): Date | undefined {
  if (choice !== "custom") return add(now, EXPIRY_PERIODS.find((p) => p.key === choice)?.period ?? {});
  if (!picked) return undefined;
  return set(picked, { hours: now.getHours(), minutes: now.getMinutes(), seconds: now.getSeconds() });
}
