/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useRef, useState, type RefObject } from "react";
import { useNavigate } from "react-router";
// nerve imports
import type { IntakeTriage, IssueDetail } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { generateWorkItemLink } from "@nerve/utils";
// components
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { problemCode } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
// store
import { effectiveStatus, inIntake } from "@/store/issue/intake/status";
// local imports
import type { IntakeLanding } from "../../landing";
import { useIntakePageState } from "../../page-state";
import { useDetailFocus } from "./detail-focus";

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
type OpenModal = { name: IntakeModal; detail: IssueDetail; accepted?: boolean };

/** The action model the desktop and the narrow header read alike, and the modals the detail region draws. */
export type IntakeActions = {
  /** The actions offered on the work item as held, in the headers' order; none while it is not held. */
  offered: readonly IntakeAction[];
  /** The status's primary action (R35): accepting, for an admin on a pending or snoozed work item; none otherwise. */
  primary: IntakeAction | undefined;
  /** The work item's address, by its project's identifier: what copy copies and open opens. */
  link: string;
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
  /** Deletes the work item: once nerve confirms, the selection leaves it for none; rejects on a refusal, which is said. */
  remove: () => Promise<void>;
};

/**
 * What the caller is offered on a work item of a project's intake (P5a spec 2.8, R11, R15, R20, R27): an archived
 * project's, or one triaged already (accepted, declined, a duplicate), only its copy and its opening; else an admin
 * (a workspace admin who is the project's member too, as the permission store gives it) triages it, a snooze offering
 * its end and another day instead of a snooze, and deletes it; a member who submitted it deletes it; anyone else, a
 * guest who submitted it too, only copies and opens it. A snooze whose day has come is pending (effectiveStatus).
 */
function offeredOn(
  detail: IssueDetail | undefined,
  role: EUserPermissions | undefined,
  me: string | undefined,
  archived: boolean
): IntakeAction[] {
  if (!detail) return [];
  const status = detail.intake ? effectiveStatus(detail.intake, new Date()) : undefined;
  if (archived || (status !== "pending" && status !== "snoozed")) return ["copy", "open"];
  if (role === EUserPermissions.ADMIN)
    return [
      "accept",
      "decline",
      ...(status === "snoozed" ? (["unsnooze", "resnooze"] as const) : (["snooze"] as const)),
      "duplicate",
      "copy",
      "open",
      "delete",
    ];
  if (role === EUserPermissions.MEMBER && me !== undefined && detail.created_by_id === me)
    return ["copy", "open", "delete"];
  return ["copy", "open"];
}

/**
 * The action model of the selected work item of a project's intake (P5a spec 2.7, 2.8, R14, R29, R35): which actions
 * the caller is offered, by his role in the route's project, the work item's status and the project, and what each
 * does. Its triages and its deletion go through the work item's queue of writes (root.writes.triageIntake,
 * deleteIntake), one out at a time (busy), each followed only in the detail's scope (the same selection, session and
 * permissions): a refusal changes nothing and is said; nerve's conflict closes the modal, the selection kept, read
 * again (issue.intake_transition_not_allowed), or left for none (issue.intake_not_found: gone from the intake). Once
 * nerve confirms a triage, the selection leaves the work item (selection.leave) for the row after it in the list the
 * page shows, else the row before, as the triage began (a duplicate aside, which stays); the acceptance's, once its
 * modal closes, its update confirmed or cancelled. Deleting leaves the work item gone, which the detail region is told
 * (leaving) before the delete is sent; once nerve confirms, the selection leaves it for none. The focus lands as each
 * ends (useDetailFocus, and the page's landing as the selection leaves).
 */
export function useIntakeActions(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  selection: {
    leave: IntakeLanding["leave"];
    leaving: RefObject<boolean>;
    region: RefObject<HTMLElement | null>;
  }
): IntakeActions {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const copyLink = useCopyLink();
  const refusalToast = useRefusalToast();
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const {
    rootIssueStore: root,
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const { filters } = useIntakePageState();
  const { data: currentUser } = useUser();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getProjectById } = useProject();
  const focus = useDetailFocus(selection.region);
  const [modal, setModal] = useState<OpenModal>();
  const [busy, setBusy] = useState(false);
  // the modal as a command's follow-up finds it, which a render may not show yet
  const open = useRef<OpenModal>(undefined);
  // the row the selection leaves for once the accept modal closes, taken as the acceptance began
  const acceptedFor = useRef<string | undefined>(undefined);
  // derived values
  const detail = getIssueDetailById(issueId);
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const offered = offeredOn(detail, role, currentUser?.id, !!project?.archived_at);
  const link = generateWorkItemLink({
    workspaceSlug,
    projectId,
    issueId,
    projectIdentifier: project?.identifier,
    sequenceId: detail?.sequence_id,
  });
  const scope = () => root.scope(workspaceSlug, projectId);
  const show = (next: OpenModal | undefined) => {
    open.current = next;
    setModal(next);
  };
  /** The row after the work item in the list the page shows, else the row before; none for the only one (R14). */
  const neighbour = (): string | undefined => {
    const ids = root.intake.getList(projectId, filters)?.data.map((row) => row.id) ?? [];
    const at = ids.indexOf(issueId);
    return at === -1 ? undefined : (ids[at + 1] ?? ids[at - 1]);
  };
  /** nerve's conflict (R28): the modal closes, the selection kept, or left for none for a work item gone. */
  const conflicted = (error: unknown): boolean => {
    const code = problemCode(error);
    if (code !== "issue.intake_transition_not_allowed" && code !== "issue.intake_not_found") return false;
    show(undefined);
    if (code === "issue.intake_not_found") selection.leave(issueId, undefined);
    else focus.land("primary");
    return true;
  };
  /**
   * Sends a write of the work item, one out at a time (busy), followed in the detail's scope only: once nerve
   * confirms, resolves; a refusal, refused, then rejects; out of the scope, rejects as a change of session.
   */
  const command = async (send: () => Promise<void>, refused: (error: unknown) => void) => {
    const current = capture();
    const inSession = sessionGuard();
    const followed = () => current() && inSession();
    if (!followed()) throw new SessionChangedError();
    setBusy(true);
    try {
      await send();
    } catch (error) {
      if (followed() && !(error instanceof SessionChangedError)) {
        setBusy(false);
        refused(error);
      }
      throw error;
    }
    if (!followed()) throw new SessionChangedError();
    setBusy(false);
  };
  const triage = async (body: IntakeTriage, underField?: (error: unknown) => boolean) => {
    const to = neighbour();
    await command(
      () => root.writes.triageIntake(issueId, body, scope()),
      (error) => {
        if (!underField?.(error)) refusalToast(error);
        // a refusal of an action no modal holds gives the focus back to the control that took it
        if (!conflicted(error) && !open.current) focus.land("opener");
      }
    );
    show(undefined);
    if (body.status === "duplicate") focus.land("primary");
    else selection.leave(issueId, to);
  };
  const take = (action: IntakeAction, opener?: HTMLElement | null) => {
    if (!offered.includes(action)) return;
    focus.remember(opener);
    if (action === "copy") void copyLink(link, t("common.link_copied_to_clipboard"));
    else if (action === "open") void navigate(link);
    else if (action === "unsnooze") void triage({ status: "pending" }).catch(() => undefined);
    else if (detail) {
      acceptedFor.current = undefined;
      show({ name: action === "resnooze" ? "snooze" : action, detail });
    }
  };
  return {
    offered,
    primary: offered.includes("accept") ? "accept" : undefined,
    link,
    modal,
    busy,
    take,
    close: () => {
      const closing = open.current;
      if (!closing) return;
      show(undefined);
      if (closing.accepted) selection.leave(issueId, acceptedFor.current);
      else focus.land("opener");
    },
    accept: async () => {
      // R29: accepted by an earlier submit of this modal, or held out of the intake (by another's), it is not triaged
      // again; one not held, a write's read again out, is triaged: nerve refuses it as a conflict if another accepted it
      const held = getIssueDetailById(issueId);
      if (!open.current?.accepted && (held === undefined || inIntake(held))) {
        acceptedFor.current = neighbour();
        await command(
          () => root.writes.triageIntake(issueId, { status: "accepted" }, scope()),
          (error) => {
            refusalToast(error);
            conflicted(error);
          }
        );
      } else acceptedFor.current ??= neighbour();
      if (open.current) show({ ...open.current, accepted: true });
    },
    triage,
    remove: async () => {
      selection.leaving.current = true;
      try {
        await command(() => root.writes.deleteIntake(issueId, scope()), refusalToast);
      } catch (error) {
        selection.leaving.current = false;
        throw error;
      }
      show(undefined);
      selection.leave(issueId, undefined);
    },
  };
}
