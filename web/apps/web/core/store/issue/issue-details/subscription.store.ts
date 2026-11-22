/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observable, runInAction } from "mobx";
import type { IssueDetail } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IIssueDetail } from "./root.store";

/**
 * The caller's subscriptions to work items (M7 design 3.10, P4 spec 2.3, R13, R14). Whether he subscribes to one is
 * its detail's is_subscribed as nerve last gave it, never set here: a value made here would go through the work item's
 * reconciliation against a read out meanwhile, which may keep either whole (issue.store.ts choose). A change is sent in
 * the work item's queue, after the writes of it already out (enqueueIssueChanges); whether nerve made it or refused it,
 * the work item is read again in that turn (refreshAuthoritative), whose answer says whether he subscribes now; a
 * refusal then fails the change. A failure of that read, nerve having made the change, is the work item's own (its
 * read's error, which its page shows with a retry); the change is made. While a change of a work item is queued, out
 * or being read again, its subscription is pending.
 */
export class IssueSubscriptionStore {
  /** The work items a change of whose subscription is pending, by id: how many changes of each. */
  private readonly pending = observable.map<string, number>();
  constructor(private readonly rootIssueDetailStore: IIssueDetail) {}
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }

  /**
   * @description the work item's detail while the caller may change his subscription to it (R14): he has a role in
   * its project, a guest's too (nerve lets whoever reads it subscribe, M7 design 3.10); its detail is held (it alone
   * says is_subscribed); neither it nor its project is archived (nerve allows it, the page offers it not). Else
   * undefined. The one gate of the detail's button, the peek's and the palette's command.
   */
  subscribable = (ws: string, project: string, id: string): IssueDetail | undefined => {
    const detail = this.root.issues.getIssueDetailById(id);
    if (!detail || detail.project_id !== project || detail.archived_at) return undefined;
    const rootStore = this.root.rootStore;
    if (rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(ws, project) === undefined)
      return undefined;
    return rootStore.projectRoot.project.getProjectById(project)?.archived_at ? undefined : detail;
  };

  /** @description whether a change of the work item's subscription is queued, out or being read again */
  isPending = (id: string): boolean => (this.pending.get(id) ?? 0) > 0;

  /** @description subscribes the caller to the work item; settles once nerve's read of it again has */
  subscribe = (ws: string, project: string, id: string): Promise<void> =>
    this.change(ws, project, id, () => this.root.detailApi.subscribe(id));

  /** @description unsubscribes the caller from the work item; settles once nerve's read of it again has */
  unsubscribe = (ws: string, project: string, id: string): Promise<void> =>
    this.change(ws, project, id, () => this.root.detailApi.unsubscribe(id));

  private change(ws: string, project: string, id: string, send: () => Promise<void>): Promise<void> {
    this.count(id, 1);
    return this.root.writes
      .enqueueIssueChanges(
        [id],
        async () => {
          const refused = await send().then(
            () => undefined,
            (error: unknown) => ({ error })
          );
          await this.root.issues.refreshAuthoritative(id, { afterWrite: true }).catch((error: unknown) => {
            if (error instanceof SessionChangedError) throw error;
          });
          if (refused) throw refused.error;
        },
        { workspaceSlug: ws, projectID: project }
      )
      .finally(() => this.count(id, -1));
  }

  private count(id: string, by: 1 | -1) {
    runInAction(() => {
      const left = (this.pending.get(id) ?? 0) + by;
      if (left > 0) this.pending.set(id, left);
      else this.pending.delete(id);
    });
  }
}
