/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState, type RefObject } from "react";
import { useNavigate } from "react-router";
// nerve imports
import type { IntakeTriage, IssueDetail } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { generateWorkItemLink } from "@nerve/utils";
// components
import { useDetailCommand } from "@/components/issues/issue-detail/use-issue-operations";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
// store
import { effectiveStatus } from "@/store/issue/intake/status";

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

/** The action model the desktop and the narrow header read alike, and the modals the detail region draws. */
export type IntakeActions = {
  /** The actions offered on the work item as held, in the headers' order; none while it is not held. */
  offered: readonly IntakeAction[];
  /** The status's primary action (R35): accepting, for an admin on a pending or snoozed work item; none otherwise. */
  primary: IntakeAction | undefined;
  /** The work item's address, by its project's identifier: what copy copies and open opens. */
  link: string;
  /** The modal open, with the work item as held when it opened. */
  modal: { name: IntakeModal; detail: IssueDetail } | undefined;
  /** Takes an offered action: opens its modal, or does it (copy, open, unsnooze). */
  take: (action: IntakeAction) => void;
  /** Closes the modal. */
  close: () => void;
  /** Accepts the work item, the accept modal's beforeFormSubmit: rejects on a refusal, so that no update follows. */
  accept: () => Promise<void>;
  /** Triages the work item: closes the modal once nerve confirms; a refusal is said and leaves the modal open. */
  triage: (body: IntakeTriage) => Promise<void>;
  /** Deletes the work item: clears the selection once nerve confirms; rejects on a refusal, which is said. */
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
 * The action model of the selected work item of a project's intake (P5a spec 2.7, 2.8): which actions the caller is
 * offered, by his role in the route's project, the work item's status and the project, and what each does. Its
 * triage and deletion go through the work item's queue of writes (root.writes.triageIntake, deleteIntake), each in the
 * detail's scope (useDetailCommand): a refusal is said by its toast and changes nothing; an answer that comes after the
 * selection moved on, or after the session or the caller's permissions changed, is followed by nothing. Deleting
 * leaves the work item gone, which the detail region is told (leaving) before the delete is sent, so that it does not
 * take it for one it cannot read; once nerve confirms, the selection is cleared (clearSelection).
 */
export function useIntakeActions(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  selection: { clearSelection: () => void; leaving: RefObject<boolean> }
): IntakeActions {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const copyLink = useCopyLink();
  const run = useDetailCommand(workspaceSlug, projectId, issueId);
  const {
    rootIssueStore: root,
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const { data: currentUser } = useUser();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getProjectById } = useProject();
  const [modal, setModal] = useState<IntakeActions["modal"]>();
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
  const send = (body: IntakeTriage) =>
    run(() => root.writes.triageIntake(issueId, body, root.scope(workspaceSlug, projectId)));
  const triage = (body: IntakeTriage) =>
    send(body).then(
      () => setModal(undefined),
      // said by its toast, or not followed (useDetailCommand)
      () => undefined
    );
  const open = (name: IntakeModal) => {
    if (detail) setModal({ name, detail });
  };
  const take = (action: IntakeAction) => {
    if (!offered.includes(action)) return;
    if (action === "copy") void copyLink(link, t("common.link_copied_to_clipboard"));
    else if (action === "open") void navigate(link);
    else if (action === "unsnooze") void triage({ status: "pending" });
    else open(action === "resnooze" ? "snooze" : action);
  };
  return {
    offered,
    primary: offered.includes("accept") ? "accept" : undefined,
    link,
    modal,
    take,
    close: () => setModal(undefined),
    accept: () => send({ status: "accepted" }),
    triage,
    remove: async () => {
      selection.leaving.current = true;
      try {
        await run(() => root.writes.deleteIntake(issueId, root.scope(workspaceSlug, projectId)));
      } catch (error) {
        selection.leaving.current = false;
        throw error;
      }
      selection.clearSelection();
    },
  };
}
