/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState, type ComponentProps } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import type { DateRange } from "@nerve/propel/calendar";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { useModuleChange } from "./use-module-change";

type Props = {
  module: Module;
  /** Whether the caller changes the module's days here: he writes its project's modules, and it is not archived. */
  editable: boolean;
} & Pick<
  ComponentProps<typeof DateRangeDropdown>,
  "buttonContainerClassName" | "buttonVariant" | "className" | "hideIcon" | "mergeDates"
>;

/**
 * A module's days, calendar days (P6a spec 2.6, R13), in its sidebar, its row and its card, which the caller changes
 * there while editable. A range goes to nerve as a pair, once both its days are picked or both cleared (a day picked
 * alone waits for the other: one day alone is the dialog's to set); the days the module has go nowhere. It is nerve's
 * change to make (useModuleChange): while the pair is out, the days picked show and the picker is busy; refused, the
 * module's days show again.
 */
export const ModuleDates = observer(function ModuleDates({ module, editable, ...look }: Props) {
  // hooks
  const { t } = useTranslation();
  const { update } = useModules();
  const { busy, send } = useModuleChange();
  // states
  const [picked, setPicked] = useState<DateRange>();

  const change = async (start_date: string | null, target_date: string | null) => {
    if (await send(() => update(module.id, { start_date, target_date }))) setPicked(undefined);
  };

  const select = (range: DateRange | undefined) => {
    const start = renderFormattedPayloadDate(range?.from) ?? null;
    const target = renderFormattedPayloadDate(range?.to) ?? null;
    if (start === module.start_date && target === module.target_date) {
      setPicked(undefined);
      return;
    }
    setPicked(range ?? { from: undefined });
    if ((start === null) === (target === null)) void change(start, target);
  };

  const shown = picked ?? { from: getDate(module.start_date), to: getDate(module.target_date) };
  return (
    <DateRangeDropdown
      {...look}
      value={{ from: shown.from, to: shown.to }}
      onSelect={select}
      placeholder={{ from: t("start_date"), to: t("end_date") }}
      disabled={!editable || busy}
    />
  );
});
