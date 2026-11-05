/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { AssetService } from "@/services/asset.service";
import type { PagedContent } from "./paged-content";
/** A missing response requires read-only verification before another explicit deletion attempt. */
export class ContentDeletion {
  private readonly pending = new Map<string, Promise<void>>();
  private readonly uncertain = observable.set<string>();
  private readonly verified = new Set<string>();
  private readonly controller = new AbortController();
  private ended = false;
  constructor(
    private readonly page: Pick<
      PagedContent<{ id: string }>,
      "view" | "current" | "rows" | "confirmRemoved" | "refreshConfirmed" | "load"
    >,
    private readonly service: AssetService,
    private readonly confirmed: (id: string) => void,
    private readonly refresh: () => Promise<void>
  ) {
    makeObservable<ContentDeletion, "ended">(this, { ended: observable });
    page.view.signal.addEventListener("abort", this.end, { once: true });
  }
  private end = () => this.dispose();
  get uncertainIDs(): readonly string[] {
    return this.current ? [...this.uncertain] : [];
  }
  private get current() {
    return !this.ended && this.page.current;
  }
  remove(id: string): Promise<void> {
    const pending = this.pending.get(id);
    if (pending) return pending;
    const request = Promise.resolve()
      .then(() => this.perform(id))
      .finally(() => {
        if (this.pending.get(id) === request) this.pending.delete(id);
      });
    this.pending.set(id, request);
    return request;
  }
  private async perform(id: string): Promise<void> {
    if (!this.current || !this.page.view.canRemoveAttachment()) throw new SessionChangedError();
    if (!this.page.rows.some((row) => row.id === id) && !this.uncertain.has(id))
      throw new Error("Attachment has not been read");
    if (this.uncertain.has(id) && !this.verified.has(id))
      throw new Error("Read the uncertain deletion before trying again");
    this.verified.delete(id);
    try {
      await this.service.remove(id, this.controller.signal);
    } catch (error) {
      if (!this.current) return;
      if (!(error instanceof ApiError) || error.status >= 500)
        runInAction(() => {
          this.uncertain.add(id);
        });
      throw error;
    }
    if (!this.current) return;
    this.page.confirmRemoved(id);
    runInAction(() => {
      this.uncertain.delete(id);
    });
    if (!this.current) return;
    this.confirmed(id);
    if (this.current) await this.page.refreshConfirmed(this.refresh);
  }
  async verify(): Promise<void> {
    await this.page.load();
    if (!this.current) return;
    await Promise.all(
      [...this.uncertain].map(async (id) => {
        try {
          await this.service.read(id, this.controller.signal);
          if (this.current) this.verified.add(id);
        } catch (error) {
          if (!this.current) return;
          if (error instanceof ApiError && error.status === 404) {
            this.verified.add(id);
            return;
          }
          throw error;
        }
      })
    );
  }
  dispose(): void {
    if (this.ended) return;
    runInAction(() => {
      this.ended = true;
      this.uncertain.clear();
    });
    this.controller.abort();
    this.verified.clear();
    this.page.view.signal.removeEventListener("abort", this.end);
  }
}
