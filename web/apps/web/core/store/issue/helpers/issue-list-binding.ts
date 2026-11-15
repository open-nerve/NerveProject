/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { makeObservable, observable, runInAction } from "mobx";
import type { TLoader } from "@nerve/types";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { Reconciled } from "@/lib/reconciled";
import {
  confirmGroups,
  mergePages,
  pagePartKey,
  type ConfirmedIssueChange,
  type ListContext,
  type PageAnswer,
  type PagePart,
  type PagedGroups,
  type Group,
} from "@/lib/paged-groups";
import { replayGroups } from "@/lib/paged-groups.replay";
import { issueParts } from "@/lib/issue-membership";
import type { VerificationRound } from "../issue-verification";
import type { IIssueRootStore, IssueWriteScope } from "../root.store";
import { listCatalog, prepareListCatalog } from "./issue-list-catalog";
import { allowsRole, readPage, scopeRole, type IssueListScope, type ReadRequest } from "./issue-list-read";

/** One immutable scope/query, retained when another list is selected. */
export class IssueListBinding {
  readonly reconciled = new Reconciled<PagedGroups>();
  readonly loaders = observable.map<string, TLoader>();
  private readonly loading = new Map<string, symbol>();
  private permissionStale = false;
  private source = 0;
  private visibleParts: readonly PagePart[] = [];
  get isVisible(): boolean {
    return this.visibleParts.length > 0;
  }
  get verifying(): boolean {
    return !this.permissionStale && this.root.verification.isVerifying(this);
  }
  setVisibleParts(parts: readonly PagePart[]): void {
    this.visibleParts = parts;
  }
  private readonly failedReads = new Map<string, ReadRequest>();
  error: unknown;
  constructor(
    readonly root: IIssueRootStore,
    readonly scope: IssueListScope
  ) {
    makeObservable<this, "permissionStale">(this, { error: observable.ref, permissionStale: observable });
    for (const value of Object.values(scope.query)) if (Array.isArray(value)) Object.freeze(value);
    Object.freeze(scope.query);
    Object.freeze(scope);
    root.registerList(this);
  }
  get context(): ListContext {
    return {
      ...this.scope,
      currentUserID: this.root.currentUserId ?? "",
      pendingIDs: this.root.writes.pendingIDs,
      catalog: listCatalog(this.root),
    };
  }
  covers(scope: IssueWriteScope): boolean {
    return (
      this.scope.workspaceID === scope.workspaceID &&
      (this.scope.scope === "project"
        ? this.scope.projectID === scope.projectID
        : !this.scope.query.project_id?.length || this.scope.query.project_id.includes(scope.projectID))
    );
  }
  invalidatePermission(): void {
    this.reconciled.invalidate();
    this.failedReads.clear();
    this.loading.clear();
    runInAction(() => {
      this.permissionStale = true;
      this.error = undefined;
      this.loaders.clear();
    });
  }
  get value(): PagedGroups | undefined {
    return this.permissionStale ? undefined : this.reconciled.value;
  }
  private read(request: ReadRequest): Promise<PageAnswer> {
    return readPage(this, request, () => ++this.source);
  }
  prepareCatalog(issues: Issue[]): Promise<void> {
    return prepareListCatalog(this.root, this.scope, issues);
  }
  private async fetch(
    part: PagePart | null,
    first: boolean,
    retry?: ReadRequest,
    refreshing?: VerificationRound
  ): Promise<PagedGroups | undefined> {
    this.root.assertActive();
    if (this.scope.scope === "project" && this.scope.empty) {
      // This is a synchronous logical result, not a server read: no token/source/answer is minted.
      if (!allowsRole(this.scope, scopeRole(this.root, this.scope))) return undefined;
      this.reconciled.invalidate();
      const value: PagedGroups = { total: 0, groups: new Map() };
      runInAction(() => {
        this.reconciled.value = value;
        this.permissionStale = false;
        this.error = undefined;
      });
      return value;
    }
    const projectIDs = this.scope.scope === "project" ? [this.scope.projectID] : (this.scope.query.project_id ?? []);
    const identity = this.root.permissionReads.identity(
      this.scope.workspaceID,
      projectIDs,
      this.scope.scope === "member"
    );
    const round = this.root.verification.begin();
    const heldGroup = part ? this.group(part.group, part.subgroup) : undefined;
    const request =
      retry ??
      structuredClone({
        part,
        first,
        cursor: first ? null : (heldGroup?.nextCursor ?? null),
        lower: first ? null : (heldGroup?.cursorBoundary ?? null),
      });
    const key = part ? pagePartKey(part) : "whole";
    const loading = Symbol(key);
    this.loading.set(key, loading);
    runInAction(() => this.loaders.set(key, first ? "init-loader" : "pagination"));
    try {
      const value = part
        ? await this.reconciled.fetchPart(
            key,
            () => this.read(request),
            (held, answer) => mergePages(held, this.answer(answer), this.context)
          )
        : this.reconciled.value === undefined || this.permissionStale
          ? await this.reconciled.fetch(() => this.read(request))
          : await this.refreshWhole(request);
      if (value === undefined) return undefined;
      if (!refreshing) await this.root.verification.drain(this, round);
      if (
        !this.root.active ||
        identity !==
          this.root.permissionReads.identity(this.scope.workspaceID, projectIDs, this.scope.scope === "member")
      )
        return undefined;
      if (part === null) this.failedReads.clear();
      else this.failedReads.delete(key);
      runInAction(() => {
        if (part === null) this.permissionStale = false;
        if (!this.failedReads.size) this.error = undefined;
      });
      return value;
    } catch (error) {
      if (this.root.active && !(error instanceof SessionChangedError))
        runInAction(() => {
          this.failedReads.set(key, request);
          this.error = error;
        });
      throw error;
    } finally {
      this.root.verification.end(round);
      if (this.root.active && this.loading.get(key) === loading) {
        this.loading.delete(key);
        runInAction(() => this.loaders.delete(key));
      }
    }
  }
  private refreshWhole(request: ReadRequest): Promise<PagedGroups | undefined> {
    this.reconciled.invalidate();
    return this.reconciled.fetchPart(
      "whole",
      () => this.read(request),
      (held, answer) => mergePages(held, this.answer(answer), this.context)
    );
  }
  private answer(value: PagedGroups): PageAnswer {
    if (!value.read) throw new Error("Missing original page metadata");
    return { ...value, read: value.read };
  }
  fetchFirst = () => this.fetch(null, true);
  retryFailed = async (): Promise<void> => {
    this.root.assertActive();
    if (this.permissionStale) {
      await this.fetchFirst();
      return;
    }
    if (await this.root.verification.retry(this)) return;
    const requests = [...this.failedReads.values()];
    if (!requests.length) {
      await this.fetchFirst();
      return;
    }
    await Promise.all(
      requests
        .filter((request) => !this.loading.has(request.part ? pagePartKey(request.part) : "whole"))
        .map((request) => this.fetch(request.part, request.first, request))
    );
  };
  fetchNext = (group = "", subgroup?: string) => {
    if (!this.group(group, subgroup)?.nextCursor || this.loading.has(pagePartKey({ group, subgroup })))
      return Promise.resolve(undefined);
    return this.fetch({ group, subgroup }, false);
  };
  refreshStale = async (visible?: readonly PagePart[]): Promise<void> => {
    const value = this.reconciled.value;
    if (!value) return;
    const parts = visible
      ? visible.filter((part) => this.group(part.group, part.subgroup)?.stale)
      : [...value.groups].flatMap(([group, data]) =>
          data.subgroups
            ? [...data.subgroups].filter(([, cell]) => cell.stale).map(([subgroup]) => ({ group, subgroup }))
            : data.stale
              ? [{ group }]
              : []
        );
    await Promise.all(parts.map((part) => this.fetch(part, true)));
  };
  acceptConfirmed(change: ConfirmedIssueChange, heldSources: ReadonlySet<number>): void {
    if (!this.root.active || (this.scope.scope === "project" && this.scope.empty)) return;
    this.reconciled.confirm((value, raw) => {
      const result = raw
        ? replayGroups(value, change, this.answer(raw), this.context)
        : confirmGroups(value, change, this.context, heldSources);
      for (const id of result.verifyIds) this.root.verification.require(id, this);
      return result.value;
    });
  }
  refreshVerification(round: VerificationRound): Promise<PagedGroups | undefined> {
    return this.fetch(null, true, undefined, round);
  }
  markVerificationStale(id?: string, error?: unknown): void {
    if (!this.root.active) return;
    if (error)
      runInAction(() => {
        this.error = error;
      });
    const issue = id ? this.root.issues.held(id) : undefined;
    if (issue) this.markMembershipStale(issue, issue);
    else
      this.reconciled.confirm((value) => {
        const next = structuredClone(value);
        for (const group of next.groups.values()) {
          group.stale = true;
          for (const cell of group.subgroups?.values() ?? []) cell.stale = true;
        }
        return next;
      });
  }
  clearVerificationError(error: unknown): void {
    if (this.error !== error) return;
    this.failedReads.clear();
    runInAction(() => {
      this.error = undefined;
    });
  }
  markMembershipStale(before: Issue | undefined, after: Issue | undefined): void {
    this.reconciled.confirm((value) => {
      const next = structuredClone(value);
      for (const part of issueParts(before, this.context).concat(issueParts(after, this.context))) {
        const group = next.groups.get(part.group);
        const cell = part.subgroup === undefined ? group : group?.subgroups?.get(part.subgroup);
        if (cell) cell.stale = true;
      }
      return next;
    });
  }
  markUnavailable(error: unknown): void {
    this.failedReads.clear();
    this.reconciled.invalidate();
    runInAction(() => {
      this.error = error;
    });
    this.reconciled.confirm((value) => {
      const next = structuredClone(value);
      for (const group of next.groups.values()) {
        group.stale = true;
        for (const cell of group.subgroups?.values() ?? []) cell.stale = true;
      }
      return next;
    });
  }
  group(group = "", subgroup?: string): Group | undefined {
    const outer = this.value?.groups.get(group);
    return subgroup === undefined ? outer : outer?.subgroups?.get(subgroup);
  }
}
