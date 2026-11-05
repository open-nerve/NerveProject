/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import type { Asset } from "@nerve/api-client";
import type { ContentView } from "./content-view";
import { matchesContent } from "./content-target";
import { SessionChangedError } from "@/lib/auth/token-manager";
/** Current-view metadata only. The form retains IDs; removal never deletes a server preparation. */
export class CommentPreparations {
  private selected: readonly string[];
  private assets: readonly Asset[] = [];
  private ended = false;
  private request = 0;
  private controller = new AbortController();
  loading = false;
  error: unknown;
  constructor(
    readonly view: ContentView,
    private readonly inspect: (id: string, signal: AbortSignal) => Promise<Asset>,
    ids: readonly string[],
    private readonly changed: (ids: string[]) => void
  ) {
    if (view.host.kind !== "new_comment" || ids.length > 100 || new Set(ids).size !== ids.length)
      throw new Error("Invalid comment preparations");
    this.selected = [...ids];
    makeObservable<CommentPreparations, "selected" | "assets" | "ended">(this, {
      selected: observable.ref,
      assets: observable.ref,
      ended: observable,
      loading: observable,
      error: observable.ref,
    });
    view.signal.addEventListener("abort", this.dispose, { once: true });
  }
  get current() {
    return !this.ended && this.view.current;
  }
  get ids(): readonly string[] {
    return this.current ? this.selected : [];
  }
  get rows(): readonly Asset[] {
    return this.current ? this.assets : [];
  }
  get unresolved(): readonly string[] {
    return this.ids.filter((id) => !this.assets.some((asset) => asset.id === id));
  }
  get ready(): boolean {
    return this.current && !this.loading && this.error === undefined && this.unresolved.length === 0;
  }
  private check(asset: Asset): void {
    if (!this.current) throw new SessionChangedError();
    if (asset.state !== "uploaded" || asset.is_bound || !matchesContent(asset, this.view.host, "attachment"))
      throw new Error("The selected file is no longer an available comment preparation");
  }
  async load(): Promise<void> {
    if (!this.current) throw new SessionChangedError();
    this.controller.abort();
    this.controller = new AbortController();
    const request = ++this.request;
    const current = () => this.current && this.request === request;
    runInAction(() => {
      this.loading = true;
      this.error = undefined;
      this.assets = [];
    });
    try {
      const assets = await Promise.all(
        this.selected.map(async (id) => {
          const asset = await this.inspect(id, this.controller.signal);
          this.check(asset);
          if (asset.id !== id) throw new Error("Mismatched preparation");
          return asset;
        })
      );
      if (current())
        runInAction(() => {
          this.assets = assets;
        });
    } catch (error) {
      if (current()) {
        runInAction(() => {
          this.error = error;
        });
        throw error;
      }
    } finally {
      if (current())
        runInAction(() => {
          this.loading = false;
        });
    }
  }
  add(asset: Asset): void {
    this.check(asset);
    if (!this.ready) throw new Error("Read the selected files first");
    if (this.selected.includes(asset.id)) return;
    if (this.selected.length >= 100) throw new Error("A comment accepts at most 100 ordinary attachments");
    runInAction(() => {
      this.selected = [...this.selected, asset.id];
      this.assets = [...this.assets, asset];
    });
    if (this.current) this.changed([...this.selected]);
  }
  remove(id: string): void {
    if (!this.current) throw new SessionChangedError();
    ++this.request;
    this.controller.abort();
    runInAction(() => {
      this.selected = this.selected.filter((value) => value !== id);
      this.assets = this.assets.filter((asset) => asset.id !== id);
      this.loading = false;
      this.error = undefined;
    });
    if (this.current) this.changed([...this.selected]);
  }
  clear(): void {
    if (!this.current) return;
    ++this.request;
    this.controller.abort();
    runInAction(() => {
      this.selected = [];
      this.assets = [];
      this.loading = false;
      this.error = undefined;
    });
    this.changed([]);
  }
  payload(): string[] {
    if (!this.ready) throw new Error("Selected files have not been verified");
    return [...this.selected];
  }
  dispose = (): void => {
    if (this.ended) return;
    runInAction(() => {
      this.ended = true;
      this.assets = [];
    });
    ++this.request;
    this.controller.abort();
    this.view.signal.removeEventListener("abort", this.dispose);
  };
}
