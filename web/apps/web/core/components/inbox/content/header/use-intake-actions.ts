/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useRef, useState, type RefObject } from "react";
import { useNavigate } from "react-router";
// nerve imports
import type { IntakeTriage } from "@nerve/api-client";
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
import { inIntake } from "@/store/issue/intake/status";
// local imports
import type { IntakeSelectionPort } from "../selection-port";
import { useDetailFocus } from "./detail-focus";
import { offeredOn, type IntakeAction, type IntakeActions, type OpenModal } from "./intake-actions";

/**
 * The action model of the selected work item of a project's intake (P5a spec 2.7, 2.8, R14, R29, R35): which actions
 * the caller is offered, by his role in the route's project, the work item's status and the project, and what each
 * does. Its triages and its deletion go through the work item's queue of writes (root.writes.triageIntake,
 * deleteIntake), one out at a time (busy), each followed only in the detail's scope (the same selection, session and
 * permissions): a refusal changes nothing and is said; nerve's conflict closes the modal, the selection kept, read
 * again (issue.intake_transition_not_allowed), or left for none (issue.intake_not_found: gone from the intake). Once
 * nerve confirms a triage, the selection leaves the work item (selection.leave) for the row its selection gave as the
 * triage began (selection.neighbour; the intake page's, the row after it in the list the page shows, else the row
 * before), a duplicate aside, which stays; the acceptance's, once its modal closes, its update confirmed or cancelled.
 * Deleting leaves the work item gone, which the detail region is told (leaving) before the delete is sent; once nerve
 * confirms, the selection leaves it for none. The focus lands as each ends (useDetailFocus, and the landing of what
 * shows the selection as it leaves). It reads the selection through its port alone (P5b R18), never a page's state.
 */
export function useIntakeActions(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  // the selection's port, and whether the detail region is deleting the work item (the region's own)
  selection: Pick<IntakeSelectionPort, "leave" | "neighbour" | "region"> & { leaving: RefObject<boolean> }
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
  /** nerve's conflict (R28): the modal closes, the selection kept, or left for none for a work item gone. */
  const conflicted = (error: unknown): boolean => {
    const code = problemCode(error);
    if (code !== "issue.intake_transition_not_allowed" && code !== "issue.intake_not_found") return false;
    show(undefined);
    if (code === "issue.intake_not_found") selection.leave(issueId, undefined);
    else focus.land("primary");
    return true;
  };
  /** A refusal of an acceptance or a deletion: said, nerve's conflict followed (conflicted). */
  const sayAndFollow = (error: unknown) => {
    refusalToast(error);
    conflicted(error);
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
    const to = selection.neighbour(issueId);
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
        acceptedFor.current = selection.neighbour(issueId);
        await command(() => root.writes.triageIntake(issueId, { status: "accepted" }, scope()), sayAndFollow);
      } else acceptedFor.current ??= selection.neighbour(issueId);
      if (open.current) show({ ...open.current, accepted: true });
    },
    triage,
    remove: async () => {
      selection.leaving.current = true;
      try {
        await command(() => root.writes.deleteIntake(issueId, scope()), sayAndFollow);
      } catch (error) {
        // the work item gone from the intake, the refusal leaves it (conflicted): its absence is still the delete's
        if (problemCode(error) !== "issue.intake_not_found") selection.leaving.current = false;
        throw error;
      }
      show(undefined);
      selection.leave(issueId, undefined);
    },
  };
}
