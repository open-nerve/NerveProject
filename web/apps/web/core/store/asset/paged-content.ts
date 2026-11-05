/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { Reconciled } from "@/lib/reconciled";
import { DetailResource } from "@/store/issue/issue-details/resource";
import type { IssueRootStore } from "@/store/issue/root.store";
import type { ContentView } from "./content-view";
export type ContentPage<T> = { data: T[]; next_cursor: string | null };
/** Native cursor pages; private drafts rely on their canonical view, never on project membership. */
export class PagedContent<T extends { id: string }> {
  private readonly held = new Reconciled<ContentPage<T>>();
  private readonly detail?: DetailResource<ContentPage<T>>;
  private readonly seen = new Set<string>();
  private request = 0;
  private controller?: AbortController;
  private pendingMore?: Promise<void>;
  private ended = false;
  loading = false;
  error: unknown;
  constructor(
    protected readonly root: IssueRootStore,
    readonly view: ContentView,
    private readonly readPage: (cursor: string | undefined, signal: AbortSignal) => Promise<ContentPage<T>>
  ) {
    if (view.host.kind !== "draft")
      this.detail = new DetailResource(root, { workspaceID: view.host.workspaceID, projectIDs: [view.host.projectID] });
    makeObservable<PagedContent<T>, "ended">(this, { loading: observable, error: observable.ref, ended: observable });
    view.signal.addEventListener("abort", this.end, { once: true });
  }
  private end = () => this.dispose();
  get current(): boolean {
    return !this.ended && this.view.current;
  }
  private get page(): ContentPage<T> | undefined {
    return this.current ? (this.detail ? this.detail.value : this.held.value) : undefined;
  }
  get rows(): readonly T[] {
    return this.page?.data ?? [];
  }
  get nextCursor(): string | null | undefined {
    return this.page?.next_cursor;
  }
  protected assertCurrent(): void {
    if (!this.current) throw new SessionChangedError();
  }
  load(): Promise<void> {
    this.seen.clear();
    this.pendingMore = undefined;
    return this.fetch();
  }
  more(): Promise<void> {
    if (this.pendingMore) return this.pendingMore;
    if (!this.nextCursor) return Promise.resolve();
    const pending = this.fetch(this.nextCursor).finally(() => {
      if (this.pendingMore === pending) this.pendingMore = undefined;
    });
    this.pendingMore = pending;
    return pending;
  }
  retryRead(): Promise<void> {
    return this.load();
  }
  private async fetch(cursor?: string): Promise<void> {
    this.assertCurrent();
    const request = ++this.request;
    this.controller?.abort();
    const controller = new AbortController();
    this.controller = controller;
    runInAction(() => {
      this.loading = true;
      this.error = undefined;
    });
    const current = () => this.current && request === this.request;
    try {
      if (!current()) return;
      const read = async () => {
        const page = await this.readPage(cursor, controller.signal);
        if (!current()) throw new SessionChangedError();
        if (
          page.next_cursor !== null &&
          (!page.next_cursor || page.next_cursor === cursor || this.seen.has(page.next_cursor))
        )
          throw new Error("Attachment pagination repeated a cursor");
        const rows = new Map((cursor ? this.rows : []).map((row) => [row.id, row]));
        for (const row of page.data) rows.set(row.id, row);
        return { data: [...rows.values()], next_cursor: page.next_cursor };
      };
      if (this.detail) await this.detail.fetch(read);
      else await this.held.fetch(read);
      if (!current()) return;
      if (cursor) this.seen.add(cursor);
    } catch (error) {
      if (!current()) return;
      runInAction(() => {
        this.error = error;
      });
      throw error;
    } finally {
      if (request === this.request) {
        this.controller = undefined;
        runInAction(() => {
          this.loading = false;
        });
      }
    }
  }
  confirmRemoved(id: string): void {
    this.assertCurrent();
    const change = (page: ContentPage<T>) => ({ ...page, data: page.data.filter((row) => row.id !== id) });
    if (this.detail) this.detail.confirm(change, this.root.captureRead());
    else this.held.confirm(change);
  }
  /** Confirmation is already published. A later read failure cannot turn it into a failed command. */
  async refreshConfirmed(read: () => Promise<unknown> = () => this.load()): Promise<void> {
    try {
      if (this.detail) {
        await this.detail.refreshAfterConfirmation(read, this.root.captureRead());
        if (this.current)
          runInAction(() => {
            this.error = this.detail!.error;
          });
      } else await read();
    } catch (error) {
      if (this.current)
        runInAction(() => {
          this.error = error;
        });
    }
  }
  dispose(): void {
    if (this.ended) return;
    runInAction(() => {
      this.ended = true;
      this.loading = false;
      this.error = undefined;
    });
    ++this.request;
    this.controller?.abort();
    this.controller = undefined;
    this.pendingMore = undefined;
    this.detail?.dispose();
    this.held.invalidate();
    this.seen.clear();
    this.view.signal.removeEventListener("abort", this.end);
  }
}
