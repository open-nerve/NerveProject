/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, reaction, runInAction } from "mobx";
import type { RootStore } from "@/store/root.store";
import type { DetailRegistration } from "@/store/issue/issue-details/resource";
export type ContentHost =
  | { kind: "issue" | "new_comment"; workspaceID: string; projectID: string; issueID: string }
  | { kind: "comment"; workspaceID: string; projectID: string; issueID: string; commentID: string }
  | { kind: "draft"; workspaceID: string; projectID: string | null; draftID: string };
export type ContentUse = "attachment" | "description";
/** Identity comes from canonical owners; permission generations belong only to IssueRootStore. */
export class ContentView implements DetailRegistration {
  private readonly controller = new AbortController();
  private ended = false;
  private readonly permission: number;
  private readonly actor: string | undefined;
  private readonly unregister: () => void;
  private readonly stopOwner: () => void;
  readonly host: Readonly<ContentHost>;
  constructor(
    private readonly root: RootStore,
    host: ContentHost
  ) {
    this.host = Object.freeze({ ...host });
    this.permission = root.issue.permissionReads.capture();
    this.actor = root.issue.currentUserId;
    makeObservable<ContentView, "ended">(this, { ended: observable });
    this.unregister = root.issue.registerResource(this);
    this.stopOwner = reaction(
      () => this.ownerMatches(),
      (matches) => {
        if (!matches) this.dispose();
      }
    );
    if (!this.current) this.dispose();
  }
  get signal() {
    return this.controller.signal;
  }
  get current(): boolean {
    if (this.ended || !this.root.issue.active || !this.ownerMatches()) return false;
    const { workspaceID, projectID, kind } = this.host;
    return kind === "draft"
      ? this.root.issue.permissionReads.currentWorkspace(this.permission, workspaceID)
      : this.root.issue.permissionReads.current(this.permission, workspaceID, [projectID], true);
  }
  private workspace() {
    return this.root.workspaceRoot.workspaces?.find((row) => row.id === this.host.workspaceID);
  }
  private ownerMatches(): boolean {
    const workspace = this.workspace();
    if (!workspace || !this.actor || this.root.issue.currentUserId !== this.actor) return false;
    const host = this.host;
    if (host.kind === "draft") {
      const draft = this.root.issue.workspaceDraftIssues.getIssueById(host.draftID, workspace.slug);
      return !!draft && draft.workspace_id === host.workspaceID && draft.project_id === host.projectID;
    }
    const issue = this.root.issue.issues.getIssueDetailById(host.issueID);
    if (!issue || issue.workspace_id !== host.workspaceID || issue.project_id !== host.projectID) return false;
    if (host.kind !== "comment") return true;
    return this.root.issue.issueDetail.comment.getCommentById(host.issueID, host.commentID)?.issue_id === host.issueID;
  }
  canWrite(_use: ContentUse): boolean {
    if (!this.current) return false;
    const host = this.host;
    if (host.kind === "draft") return true;
    const issue = this.root.issue.issues.getIssueDetailById(host.issueID);
    const project = this.root.projectRoot.project.getProjectById(host.projectID);
    if (!issue || issue.archived_at || !project || project.archived_at) return false;
    if (host.kind === "comment")
      return this.root.issue.issueDetail.comment.getCommentById(host.issueID, host.commentID)?.actor_id === this.actor;
    if (host.kind === "new_comment") return true;
    const workspace = this.workspace();
    const role =
      workspace && this.root.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, host.projectID);
    return role === 15 || role === 20;
  }
  affects(workspaceID: string, projectID?: string): boolean {
    return this.host.workspaceID === workspaceID && (!projectID || this.host.projectID === projectID);
  }
  invalidate(): void {
    // An unrelated issue deletion can notify this project; the canonical host decides whether it still exists.
    if (!this.current) this.dispose();
  }
  dispose(): void {
    if (this.ended) return;
    runInAction(() => {
      this.ended = true;
    });
    this.stopOwner();
    this.unregister();
    this.controller.abort();
  }
}
