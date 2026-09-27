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
 * @type {Array<{value: EStartOfTheWeek, label: string, i18n_label: string}>}
 * @constant
 */
export const START_OF_THE_WEEK_OPTIONS = [
  {
    value: EStartOfTheWeek.SUNDAY,
    label: "Sunday",
    i18n_label: "days.sunday",
  },
  {
    value: EStartOfTheWeek.MONDAY,
    label: "Monday",
    i18n_label: "days.monday",
  },
  {
    value: EStartOfTheWeek.TUESDAY,
    label: "Tuesday",
    i18n_label: "days.tuesday",
  },
  {
    value: EStartOfTheWeek.WEDNESDAY,
    label: "Wednesday",
    i18n_label: "days.wednesday",
  },
  {
    value: EStartOfTheWeek.THURSDAY,
    label: "Thursday",
    i18n_label: "days.thursday",
  },
  {
    value: EStartOfTheWeek.FRIDAY,
    label: "Friday",
    i18n_label: "days.friday",
  },
  {
    value: EStartOfTheWeek.SATURDAY,
    label: "Saturday",
    i18n_label: "days.saturday",
  },
];
