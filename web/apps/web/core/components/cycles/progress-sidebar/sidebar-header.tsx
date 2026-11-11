/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { ChevronRightOutline } from "@makeplane/propel/icons";
import type { Cycle } from "@nerve/api-client";
import { CYCLE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { DateRange } from "@nerve/propel/calendar";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  cycle: Cycle;
  workspaceSlug: string;
  projectId: string;
  handleClose: () => void;
};

/**
 * The head of a cycle's progress sidebar (P5a spec 2.5): its close button, the cycle's name, its status as nerve gives
 * it, and its days, which the caller changes here while he writes the project's cycles (canWriteCycles, R8) and the
 * cycle has neither ended nor been archived; nerve refuses a change of the others.
 */
export const CycleSidebarHeader = observer(function CycleSidebarHeader(props: Props) {
  const { cycle, workspaceSlug, projectId, handleClose } = props;
  // hooks
  const { t } = useTranslation();
  const { canWriteCycles } = useUserPermissions();
  // auth
  const editable =
    canWriteCycles(workspaceSlug, projectId) && cycle.status !== "completed" && cycle.archived_at === null;
  const status = CYCLE_STATUS.find((one) => one.value === cycle.status);

  return (
    <>
      <div className="sticky top-0 z-10 flex items-center justify-between bg-surface-1 pt-2">
        <div className="flex size-5 items-center justify-center">
          <button
            type="button"
            aria-label={t("close")}
            className="flex size-6 flex-shrink-0 items-center justify-center rounded-full bg-layer-3 hover:bg-layer-3-hover"
            onClick={handleClose}
          >
            <ChevronRightOutline className="size-4 text-secondary" aria-hidden="true" />
          </button>
        </div>
      </div>
      <div className="flex w-full flex-col gap-2">
        <div className="flex items-start justify-between gap-3 pt-2">
          <h4 className="w-full text-18 font-semibold break-words text-primary">{cycle.name}</h4>
          {status && (
            <span
              className="flex h-6 min-w-20 items-center justify-center truncate rounded-sm px-3 text-center text-11 font-medium whitespace-nowrap"
              style={{
                color: status.color,
                backgroundColor: `${status.color}20`,
              }}
            >
              {t(status.i18n_title)}
            </span>
          )}
        </div>
        <CycleDates cycle={cycle} editable={editable} />
      </div>
    </>
  );
});

/**
 * The cycle's days, calendar days (R6), which the caller changes while editable. A range picked goes to nerve, the two
 * days as a pair (a day picked alone waits for the other; the days the cycle has go nowhere); while it is out the days
 * picked show and no other range may be picked. Only once nerve has made the change does the sidebar say so, the days
 * then nerve's; refused (another cycle on those days, the cycle ended or archived meanwhile), it says why, and the days
 * are nerve's again (R12). It follows nerve's answer in the session it was sent in, while the sidebar is still there
 * (followInSession): late, the answer reaches the store alone.
 */
const CycleDates = observer(function CycleDates({ cycle, editable }: { cycle: Cycle; editable: boolean }) {
  // hooks
  const { t } = useTranslation();
  const { update } = useCycles();
  const toastRefusal = useRefusalToast();
  // states
  const [picked, setPicked] = useState<DateRange>();
  const [busy, setBusy] = useState(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const change = async (start_date: string, end_date: string) => {
    setBusy(true);
    await followInSession(() => update(cycle.id, { start_date, end_date }), {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_cycles.action.update.success.title"),
          message: t("project_cycles.action.update.success.description"),
        });
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    if (!mounted.current) return;
    setPicked(undefined);
    setBusy(false);
  };

  const select = (range: DateRange | undefined) => {
    const start = renderFormattedPayloadDate(range?.from);
    const end = renderFormattedPayloadDate(range?.to);
    if (start === cycle.start_date && end === cycle.end_date) {
      setPicked(undefined);
      return;
    }
    setPicked(range);
    if (start !== undefined && end !== undefined) void change(start, end);
  };

  const shown = picked ?? { from: getDate(cycle.start_date), to: getDate(cycle.end_date) };
  return (
    <div className="flex items-center gap-2">
      <DateRangeDropdown
        className="h-7"
        buttonVariant="border-with-text"
        minDate={new Date()}
        value={{ from: shown.from, to: shown.to }}
        onSelect={select}
        placeholder={{
          from: t("project_cycles.start_date"),
          to: t("project_cycles.end_date"),
        }}
        mergeDates
        disabled={!editable || busy}
      />
    </div>
  );
});
