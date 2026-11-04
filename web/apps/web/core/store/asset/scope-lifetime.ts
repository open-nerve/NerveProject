/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import type { DetailRegistration } from "@/store/issue/issue-details/resource";
import type { RootStore } from "@/store/root.store";

export type AssetReadScope = Readonly<{ workspaceID?: string; projectID?: string; member?: boolean }>;

/** A monotonic view lifetime, subscribed only through the issue root's existing permission owner. */
export class ScopeLifetime implements DetailRegistration {
  private ended = false;
  private readonly permission: number;
  private readonly unregister: () => void;
  constructor(
    private readonly root: RootStore,
    readonly scope: AssetReadScope,
    private readonly allowed: () => boolean,
    private readonly onEnd: () => void
  ) {
    makeObservable<ScopeLifetime, "ended">(this, { ended: observable });
    this.permission = root.issue.permissionReads.capture();
    this.unregister = root.issue.registerResource(this);
  }

  get current(): boolean {
    return !this.ended && this.root.issue.active && this.allowed() && this.projectPermissionCurrent();
  }

  private projectPermissionCurrent(): boolean {
    return (
      !this.scope.projectID ||
      this.root.issue.permissionReads.current(
        this.permission,
        this.scope.workspaceID,
        [this.scope.projectID],
        this.scope.member
      )
    );
  }

  affects(workspaceID: string, projectID?: string, memberOnly = false): boolean {
    return (
      this.scope.workspaceID === workspaceID &&
      (!projectID || projectID === this.scope.projectID) &&
      (!memberOnly || !!this.scope.member)
    );
  }

  invalidate(): void {
    // Ordinary issue deletion also notifies project resources. It does not retire a project's image permission.
    if (this.scope.projectID && this.current) return;
    this.dispose();
  }

  dispose(): void {
    if (this.ended) return;
    runInAction(() => {
      this.ended = true;
    });
    this.unregister();
    this.onEnd();
  }
}
