/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IntakeTriage, IssueDetail, Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
// store
import { effectiveStatus } from "@/store/issue/intake/status";
// local imports
import type { IntakeSelectionPort } from "../selection-port";

/** What the caller may do with a work item of a project's intake, from its header (P5a spec 2.8). */
export type IntakeAction =
  | "accept"
  | "decline"
  | "snooze"
  | "unsnooze"
  | "resnooze"
  | "duplicate"
  | "delete"
  | "copy"
  | "open";

/** The label of each action, each key whole (tools/i18n-keys.mjs reads it). */
export const INTAKE_ACTION_LABELS: Readonly<Record<IntakeAction, string>> = {
  accept: "inbox_issue.actions.accept",
  decline: "inbox_issue.actions.decline",
  snooze: "inbox_issue.actions.snooze",
  unsnooze: "inbox_issue.actions.unsnooze",
  resnooze: "inbox_issue.actions.resnooze",
  duplicate: "inbox_issue.actions.mark_as_duplicate",
  delete: "inbox_issue.actions.delete",
  copy: "inbox_issue.actions.copy",
  open: "inbox_issue.actions.open",
};

/** The modals the actions open, one at a time. */
type IntakeModal = "accept" | "decline" | "snooze" | "duplicate" | "delete";

/** The modal open: its name, the work item as held when it opened, and, for the accept modal, whether it accepted it. */
export type OpenModal = { name: IntakeModal; detail: IssueDetail; accepted?: boolean };

/** The action model the desktop and the narrow header read alike, and the modals the detail region draws. */
export type IntakeActions = {
  /** The actions offered on the work item as held, in the headers' order; none while it is not held. */
  offered: readonly IntakeAction[];
  /** The status's primary action (R35): accepting, for an admin on a pending or snoozed work item; none otherwise. */
  primary: IntakeAction | undefined;
  /**
   * What open opens (P5b R30): the work item's address, by its project's identifier (the browse page's, which copy
   * copies); embedded, for one in the intake (not accepted), the intake page's, selecting it in the tab that lists it.
   */
  openLink: string;
  /** The modal open. */
  modal: OpenModal | undefined;
  /** Whether a triage or a deletion of the work item is out: the headers' controls wait meanwhile. */
  busy: boolean;
  /**
   * Takes an offered action from the control opener (the focus's place once the action ends, or its menu's button):
   * opens its modal, or does it (copy, open, unsnooze).
   */
  take: (action: IntakeAction, opener?: HTMLElement | null) => void;
  /**
   * The modal's close: a cancel, the focus back on the control that opened it (on the primary action, else the
   * detail, once a read again withdrew that control); for the accept modal once it accepted the work item, its
   * update's confirmed close or a cancel, the selection leaving it (R14).
   */
  close: () => void;
  /**
   * Accepts the work item, the accept modal's beforeFormSubmit, and nothing else: one this modal accepted, or one held
   * out of the intake (another accepted it, R29), is not triaged again, one not held (a read again out) is; a refusal
   * is said and rejects, so that no update follows, a conflict closing the modal.
   */
  accept: () => Promise<void>;
  /**
   * Triages the work item from its modal or its header: once nerve confirms, the modal closes and the selection leaves
   * it (R14), but for a duplicate; a refusal rejects, said unless underField shows it under the modal's field.
   */
  triage: (body: IntakeTriage, underField?: (error: unknown) => boolean) => Promise<void>;
  /**
   * Deletes the work item: once nerve confirms, the selection leaves it for none; rejects on a refusal, which is said,
   * one saying it is gone from the intake closing the modal, the selection left for none (nerve's conflict).
   */
  remove: () => Promise<void>;
};

/**
 * What the caller is offered on a work item of a project's intake (P5a spec 2.8, R11, R15, R20, R27; P5b R15, R25;
 * T11 m2), by the project as held: one whose project is archived or has its intake turned off (nerve refuses every
 * triage and deletion then, project.archived, issue.intake_disabled; a project not held is not known to have it on, as
 * intakeEditableBy takes it), or one triaged already, only its copy; else an admin (a workspace admin who is the
 * project's member too, as the permission store gives it) triages it, a snooze offering its end and another day
 * instead of a snooze, and deletes it; a member who submitted it deletes it; anyone else, a guest who submitted it
 * too, only copies it. Its opening, after its copy, once accepted (the browse page; on the intake page, else the
 * browse page sends it back there), and on every status embedded in the notifications page (kind, its selection's)
 * while the intake is on (the intake page selecting it; turned off, that page is its disabled page, P5a R19). A snooze
 * whose day has come is pending.
 */
export function offeredOn(
  detail: IssueDetail | undefined,
  project: Project | undefined,
  role: EUserPermissions | undefined,
  me: string | undefined,
  kind: IntakeSelectionPort["kind"]
): IntakeAction[] {
  if (!detail) return [];
  const status = detail.intake ? effectiveStatus(detail.intake, new Date()) : undefined;
  const on = !!project?.intake_view;
  const open = status === "accepted" || (kind === "embed" && on) ? (["open"] as const) : [];
  if (!on || project?.archived_at || (status !== "pending" && status !== "snoozed")) return ["copy", ...open];
  if (role === EUserPermissions.ADMIN)
    return [
      "accept",
      "decline",
      ...(status === "snoozed" ? (["unsnooze", "resnooze"] as const) : (["snooze"] as const)),
      "duplicate",
      "copy",
      ...open,
      "delete",
    ];
  if (role === EUserPermissions.MEMBER && me !== undefined && detail.created_by_id === me)
    return ["copy", ...open, "delete"];
  return ["copy", ...open];
}
