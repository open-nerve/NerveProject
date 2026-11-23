/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { ChevronDownOutline, ChevronUpOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
// local imports
import { useIntakePageState } from "../../page-state";

/** Where a step goes in the list: to the row before the selected one, or after it. */
type Direction = "previous" | "next";

/**
 * Selects the row before or after the selected work item in the list the intake page shows (the tab's filters, the
 * pages read so far), the first after the last and the last before the first; for a work item the list does not show,
 * its first row as the next and its last as the previous. The user's own move: a push, as a row's press is (P5a spec
 * 2.6), the tab kept.
 */
function useStep(projectId: string, issueId: string | undefined): (direction: Direction) => void {
  const { filters } = useIntakePageState();
  const { getList } = useIntake();
  const [, setParams] = useSearchParams();
  const ids = getList(projectId, filters)?.data.map((row) => row.id) ?? [];
  return (direction) => {
    if (issueId === undefined || ids.length === 0) return;
    const at = ids.indexOf(issueId);
    const step = direction === "next" ? 1 : -1;
    const to = at === -1 ? ids.at(step === 1 ? 0 : -1) : ids[(at + step + ids.length) % ids.length];
    if (to === undefined || to === issueId) return;
    setParams((params) => {
      const next = new URLSearchParams(params);
      next.set("inboxIssueId", to);
      return next;
    });
  };
}

/** Whether a key pressed there is the field's or the editor's to take: a name, a description, a comment, a search. */
const inField = (target: EventTarget | null): boolean =>
  target instanceof Element &&
  target.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"])') !== null;

/**
 * The intake page's previous and next work items (P5a spec 2.7; HO/M3-closeout.md:65): two buttons, named in the
 * page's language, that select the row before or after the selected work item (useStep). The wide and the narrow
 * header each draw them; the keys are the page's (useIntakeArrowKeys).
 */
export const IntakeNavigation = observer(function IntakeNavigation(props: { projectId: string; issueId: string }) {
  const { t } = useTranslation();
  const step = useStep(props.projectId, props.issueId);
  return (
    <div className="flex items-center gap-x-2">
      <IconButton
        variant="secondary"
        size="lg"
        icon={ChevronUpOutline}
        aria-label={t("inbox_issue.a11y.previous")}
        onClick={() => step("previous")}
      />
      <IconButton
        variant="secondary"
        size="lg"
        icon={ChevronDownOutline}
        aria-label={t("inbox_issue.a11y.next")}
        onClick={() => step("next")}
      />
    </div>
  );
});

/**
 * ↑ and ↓ select the row before or after the selected work item (useStep, P5a spec 2.7), the page's keys, once for
 * the page however many headers draw the buttons. Only while the focus is outside the fields and the editors, whose
 * keys they are (inField); never a key a control has handled (a menu's, a list box's), one pressed with a modifier, or
 * one inside a dialog, which the page behind it does not take; with nothing selected, never.
 */
export function useIntakeArrowKeys(projectId: string, issueId: string | undefined): void {
  const step = useStep(projectId, issueId);
  useEffect(() => {
    if (issueId === undefined) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return;
      if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
      if (inField(event.target)) return;
      if (event.target instanceof Element && event.target.closest('[role="dialog"]') !== null) return;
      event.preventDefault();
      step(event.key === "ArrowUp" ? "previous" : "next");
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [issueId, step]);
}
