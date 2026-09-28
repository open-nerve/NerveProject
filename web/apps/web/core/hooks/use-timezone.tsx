/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
import type { Timezone } from "@nerve/api-client";
// services
import { TimezoneService } from "@/services/timezone.service";

const timezoneService = new TimezoneService();

// One option for each time zone: the labels of the places that share it, together.
const groupTimezones = (timezones: Timezone[]): Timezone[] => {
  const grouped = new Map<string, Timezone>();
  for (const timezone of timezones) {
    const existing = grouped.get(timezone.value);
    grouped.set(timezone.value, existing ? { ...existing, label: `${existing.label}, ${timezone.label}` } : timezone);
  }
  return Array.from(grouped.values());
};

// An option's content: the offset, then the places.
const timezoneLabel = (timezone: Timezone) => (
  <div className="flex gap-1.5">
    <span className="text-placeholder">{timezone.utc_offset}</span>
    <span className="text-secondary">{timezone.label}</span>
  </div>
);

const useTimezone = () => {
  // fetching the timezone from the server
  const {
    data: timezones,
    isLoading: timezoneIsLoading,
    error: timezonesError,
  } = useSWR("TIMEZONES_LIST", () => timezoneService.list(), {
    refreshInterval: 0,
  });

  // derived values
  const isDisabled = timezoneIsLoading || timezonesError || !timezones;

  const options = [
    ...groupTimezones(timezones ?? []).map((timezone) => ({
      value: timezone.value,
      query: `${timezone.value} ${timezone.label}, ${timezone.gmt_offset}, ${timezone.utc_offset}`,
      content: timezoneLabel(timezone),
    })),
    {
      value: "UTC",
      query: "utc, coordinated universal time",
      content: "UTC",
    },
  ];

  const selectedTimezone = (value: string | undefined) => options.find((option) => option.value === value)?.content;

  return {
    timezones: options,
    isLoading: timezoneIsLoading,
    error: timezonesError,
    disabled: isDisabled,
    selectedValue: selectedTimezone,
  };
};

export default useTimezone;
