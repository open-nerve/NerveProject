/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueDetail as IssueDetailAnswer } from "@nerve/api-client";
import { problemCode } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { WritesRoot } from "./issue-write-confirm";
import type { IssueWriteScope } from "./root.store";

/** nerve's refusals of a write of a work item another triaged first (R28): it is read again in the refused write's turn. */
const TRIAGED_ELSEWHERE: ReadonlySet<string> = new Set(["issue.intake_transition_not_allowed", "issue.intake_closed"]);
/**
 * nerve's refusals of a content change or a deletion that read the work item again in the refused write's turn: those
 * of one triaged first, and of one gone from its intake (deleted, or its intake row), whose page follows canonical's
 * read (P5a spec 2.4). A triage refused as gone is not read again: its page leaves the work item on that refusal.
 */
const CHANGED_ELSEWHERE: ReadonlySet<string> = new Set([...TRIAGED_ELSEWHERE, "issue.intake_not_found"]);

/**
 * The writes of a work item of an intake (P5a spec 2.4, R3, R28, R34; re-94 S1), beside write()
 * (issue-write-confirm.ts): sent by request on the intake's route and confirmed to canonical and the intake lists. The
 * writes' unit (issue-writes.ts) holds it and sends them in the work item's queue, one at a time with its other writes.
 */
export class IssueWriteIntake {
  constructor(private readonly root: WritesRoot) {}
  /**
   * A content change of a work item in its project's intake, sent by request on the intake's route (P5a spec 2.4):
   * canonical takes nerve's answer, its detail's activity and comments are read again, its intake lists take it; the
   * counts stay as they are.
   */
  edit(id: string, scope: IssueWriteScope, request: () => Promise<IssueDetailAnswer>): Promise<IssueDetailAnswer> {
    return this.intake(id, scope, request, CHANGED_ELSEWHERE, async (answer, permission) => {
      this.root.issues.confirmChange({ id, beforeSnapshot: { kind: "unknown" }, after: answer }, permission);
      this.root.intake.confirm(answer);
      await this.root.issueDetail.refreshAfterChange(scope.workspaceSlug, scope.projectID, id);
    });
  }
  /**
   * A triage of a work item of an intake, sent by request on the intake's route (P5a spec 2.4, R34): its intake lists
   * take nerve's answer, canonical never does (a triage but an acceptance moves no updated_at, and canonical takes an
   * equal read, so a read sent before would undo it); the work item is read again from after it (readAgain), and the
   * counts are fetched again. Gives that read, or undefined for none.
   */
  async triage(
    id: string,
    scope: IssueWriteScope,
    request: () => Promise<IssueDetailAnswer>
  ): Promise<IssueDetailAnswer | undefined> {
    let read: IssueDetailAnswer | undefined;
    await this.intake(id, scope, request, TRIAGED_ELSEWHERE, async (answer) => {
      this.root.intake.confirm(answer);
      read = await this.readAgain(id, scope);
      this.countAgain(scope);
    });
    return read;
  }
  /**
   * A deletion of a work item of an intake, sent by request on the intake's route (P5a spec 2.4, R34): its intake row is
   * gone, so it leaves every intake list; then it is read again in the turn (readAgain), as nerve deletes the intake row
   * alone of one accepted meanwhile. Only a read that found none (the verification's missing, not a read overtaken or
   * undecided, which leaves canonical to the read that overtook it) deletes it from canonical; one read takes its place
   * there. No project list is touched; the counts are fetched again. A work item in an intake is no parent and no child
   * (M7 design), so none goes with it.
   */
  async delete(id: string, scope: IssueWriteScope, request: () => Promise<null>): Promise<void> {
    await this.intake(id, scope, request, CHANGED_ELSEWHERE, async (_, permission) => {
      this.root.intake.drop(scope.projectID, id);
      if (!(await this.readAgain(id, scope)) && this.root.verification.isMissing(id))
        this.root.issues.confirmChange({ id, beforeSnapshot: { kind: "unknown" }, after: null }, permission);
      this.countAgain(scope);
    });
  }
  /**
   * The intake's route (P5a spec 2.4, R3; re-94 S1): a write of a work item of an intake, beside write(), with its
   * permission marks alone. It reads no project list's directory before it is sent or as it is confirmed (the work
   * item is in its project's triage state, which no directory lists, so a list that needs the states would fail it),
   * confirms no project list, makes none unavailable or read again (issueMembership does not read intake: one could
   * take it for a row), and keeps no pending write (only project lists read one). nerve's answer, from the permission
   * scope it was sent from, is confirmed by confirm; a refusal is followed (refused, reading the work item again on the
   * codes readAgainOn names) and handed on. Gives the answer.
   */
  private async intake<T extends IssueDetailAnswer | null>(
    id: string,
    scope: IssueWriteScope,
    request: () => Promise<T>,
    readAgainOn: ReadonlySet<string>,
    confirm: (answer: T, permission: number) => Promise<void> | void
  ): Promise<T> {
    const root = this.root;
    root.assertActive();
    const permission = root.permissionReads.capture();
    let answer: T;
    try {
      answer = await request();
    } catch (error) {
      await this.refused(id, scope, error, readAgainOn);
      throw error;
    }
    root.assertActive();
    if (answer && answer.workspace_id !== scope.workspaceID) throw new SessionChangedError();
    // The write succeeded, but an answer from an invalidated permission scope cannot revise what is held.
    if (root.permissionReads.current(permission, scope.workspaceID, [scope.projectID], true))
      await confirm(answer, permission);
    return answer;
  }
  /**
   * What a refusal of a write of a work item of an intake changes, in the write's turn, before it is handed on (P5a
   * spec 2.4, R28; re-94 m1): a work item gone from its intake (issue.intake_not_found) leaves every intake list; on a
   * code readAgainOn names it is read again (readAgain), its read confirmed to its intake lists, so that canonical
   * holds nerve's read (one gone missing, one out of the intake without its intake row); the counts are fetched again
   * after either. Any other refusal leaves every value as it was.
   */
  private async refused(
    id: string,
    scope: IssueWriteScope,
    error: unknown,
    readAgainOn: ReadonlySet<string>
  ): Promise<void> {
    this.root.assertActive();
    const code = problemCode(error);
    const gone = code === "issue.intake_not_found";
    if (gone) this.root.intake.drop(scope.projectID, id);
    if (code !== undefined && readAgainOn.has(code)) {
      const read = await this.readAgain(id, scope);
      if (read) this.root.intake.confirm(read);
    } else if (!gone) return;
    this.countAgain(scope);
  }
  /**
   * Reads the work item again in its write's turn, from after the write (P5a R34, R28): a read sent before is not
   * taken, and canonical takes nerve's read. A work item read as gone, or by a read overtaken and undecided
   * (undefined), leaves every intake list. A failure but a change of session stays the work item's own (its
   * verification keeps it, for the detail to show and retry) and fails no write, which nerve made
   * (subscription.store.ts). Gives the read, or undefined for none.
   */
  private async readAgain(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer | undefined> {
    let read: IssueDetailAnswer | undefined;
    try {
      read = await this.root.issues.refreshAuthoritative(id, { afterWrite: true });
    } catch (error) {
      if (error instanceof SessionChangedError) throw error;
      return undefined;
    }
    if (read === undefined) this.root.intake.drop(scope.projectID, id);
    return read;
  }
  /** Fetches the counts of the scope's workspace again, if this generation holds them (R8; IntakeStore.countAgain). */
  private countAgain(scope: IssueWriteScope): void {
    this.root.intake.countAgain({ id: scope.workspaceID, slug: scope.workspaceSlug });
  }
}
