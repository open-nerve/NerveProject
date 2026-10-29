/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Reconciled, type Change } from "@/lib/reconciled";
import { makeObservable, observable, runInAction } from "mobx";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueReadToken, IssueRootStore } from "../root.store";

export type DetailScope = Readonly<{ workspaceID: string; projectIDs: readonly string[]; member?: boolean }>;
export type DetailRead = { token: IssueReadToken; isCurrent: (projects?: readonly string[]) => boolean };
export interface DetailRegistration {
  affects(workspaceID: string, projectID?: string, memberOnly?: boolean): boolean;
  invalidate(): void;
  dispose(): void;
}

type Proven<V> = { value: V; token: IssueReadToken; projects: readonly string[]; lifetime: number };

/** A resource owns its DTO only; permissions, invalidation and session lifetime belong to the issue root. */
export class DetailResource<V> implements DetailRegistration {
  private readonly held = new Reconciled<Proven<V>>();
  private readonly unregister: () => void;
  private disposed = false;
  private lifetime = 0;
  private request = 0;
  error: unknown;
  loading = false;
  constructor(
    private readonly root: IssueRootStore,
    readonly scope: DetailScope
  ) {
    this.scope = Object.freeze({ ...scope, projectIDs: Object.freeze([...new Set(scope.projectIDs)].toSorted()) });
    makeObservable<DetailResource<V>, "lifetime" | "disposed">(this, {
      error: observable.ref,
      loading: observable,
      lifetime: observable,
      disposed: observable,
    });
    this.unregister = root.registerResource(this);
  }
  private readable(projects: readonly string[]): boolean {
    const stores = this.root.rootStore;
    const workspace = stores.workspaceRoot.workspaces?.find((value) => value.id === this.scope.workspaceID);
    return (
      !!workspace &&
      (!this.scope.member || workspace.role !== 5) &&
      projects.every(
        (projectID) =>
          stores.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, projectID) !== undefined
      )
    );
  }
  private current(token: IssueReadToken, projects: readonly string[]): boolean {
    return (
      !this.disposed && this.root.isReadCurrent(token, [...projects], this.scope.workspaceID) && this.readable(projects)
    );
  }
  private visible(held: Proven<V>): boolean {
    return held.lifetime === this.lifetime && this.current(held.token, held.projects);
  }
  get value(): V | undefined {
    const held = this.held.value;
    return held && this.visible(held) ? held.value : undefined;
  }
  async fetch(
    read: (read: DetailRead) => Promise<V>,
    projects: (value: V) => readonly string[] = () => []
  ): Promise<V | undefined> {
    if (this.disposed || !this.readable(this.scope.projectIDs)) throw new SessionChangedError();
    const token = this.root.captureRead(this.scope.member);
    const request = ++this.request;
    const lifetime = this.lifetime;
    runInAction(() => {
      this.loading = true;
      this.error = undefined;
    });
    try {
      const answer = await this.held.fetch(async () => {
        const value = await read({
          token,
          // Unknown related projects may still need directory proof; this only checks the read's generation.
          isCurrent: (involved = this.scope.projectIDs) =>
            request === this.request &&
            lifetime === this.lifetime &&
            !this.disposed &&
            this.root.isReadCurrent(token, [...involved], this.scope.workspaceID),
        });
        const involved = [...new Set([...this.scope.projectIDs, ...projects(value)])];
        if (lifetime !== this.lifetime || !this.current(token, involved)) throw new SessionChangedError();
        return { value, token, projects: involved, lifetime };
      });
      return answer && this.visible(answer) ? answer.value : undefined;
    } catch (error) {
      if (request !== this.request || !this.current(token, this.scope.projectIDs)) return undefined;
      runInAction(() => {
        this.error = error;
      });
      throw error;
    } finally {
      if (request === this.request)
        runInAction(() => {
          this.loading = false;
        });
    }
  }
  confirm(change: Change<V>, token: IssueReadToken): void {
    const projects = this.held.value?.projects ?? this.scope.projectIDs;
    if (!this.current(token, projects)) return;
    this.held.confirm((held, answered) =>
      this.visible(held) ? { ...held, value: change(held.value, answered?.value) } : held
    );
  }
  affects(workspaceID: string, projectID?: string, memberOnly = false): boolean {
    return (
      this.scope.workspaceID === workspaceID &&
      (!memberOnly || !!this.scope.member) &&
      (!projectID || this.scope.projectIDs.includes(projectID) || !!this.held.value?.projects.includes(projectID))
    );
  }
  invalidate(): void {
    ++this.request;
    this.held.invalidate();
    runInAction(() => {
      ++this.lifetime;
      this.loading = false;
      this.error = undefined;
    });
  }
  dispose(): void {
    if (this.disposed) return;
    this.invalidate();
    runInAction(() => {
      this.disposed = true;
    });
    this.unregister();
  }
}
