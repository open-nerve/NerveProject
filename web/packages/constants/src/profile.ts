/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import { EStartOfTheWeek } from "@nerve/types";

export const PROFILE_TABS = [
  {
    key: "assigned",
    route: "assigned",
    i18n_label: "profile.tabs.assigned",
  },
  {
    key: "created",
    route: "created",
    i18n_label: "profile.tabs.created",
  },
  {
    key: "subscribed",
    route: "subscribed",
    i18n_label: "profile.tabs.subscribed",
  },
];

/**
 * @description The options for the start of the week
 * @type {Array<{value: EStartOfTheWeek, i18n_label: string}>}
 * @constant
 */
export const START_OF_THE_WEEK_OPTIONS = [
  {
    value: EStartOfTheWeek.SUNDAY,
    i18n_label: "days.sunday",
  },
  {
    value: EStartOfTheWeek.MONDAY,
    i18n_label: "days.monday",
  },
  {
    value: EStartOfTheWeek.TUESDAY,
    i18n_label: "days.tuesday",
  },
  {
    value: EStartOfTheWeek.WEDNESDAY,
    i18n_label: "days.wednesday",
  },
  {
    value: EStartOfTheWeek.THURSDAY,
    i18n_label: "days.thursday",
  },
  {
    value: EStartOfTheWeek.FRIDAY,
    i18n_label: "days.friday",
  },
  {
    value: EStartOfTheWeek.SATURDAY,
    i18n_label: "days.saturday",
  },
];
