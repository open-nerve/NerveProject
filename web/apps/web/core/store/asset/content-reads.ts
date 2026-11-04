/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { SessionChangedError } from "@/lib/auth/token-manager";
import { downloadAsset } from "@/lib/asset-download";
import type { AssetService } from "@/services/asset.service";
import type { ContentView } from "./content-view";
import { ContentReadEntry, type ContentRead, type ContentSnapshot } from "./content-read-entry";
export type { ContentRead, ContentSnapshot } from "./content-read-entry";
export interface ContentReadLease {
  getSnapshot(): ContentSnapshot;
  subscribe(changed: () => void): () => void;
  loaded(stamp: number): void;
  failed(stamp: number): void;
  retry(): Promise<void>;
  dispose(): void;
}
class ReadLease implements ContentReadLease {
  private disposed = false;
  private readonly ended: ContentSnapshot = { status: "unavailable", stamp: -1 };
  private readonly subscriptions = new Set<() => void>();
  constructor(
    private readonly entry: ContentReadEntry,
    private readonly onRelease: () => void
  ) {
    entry.retain();
  }
  getSnapshot = (): ContentSnapshot => (this.disposed ? this.ended : this.entry.snapshot);
  subscribe = (changed: () => void): (() => void) => {
    if (this.disposed) return () => {};
    this.subscriptions.add(changed);
    this.entry.listeners.add(changed);
    return () => {
      this.subscriptions.delete(changed);
      this.entry.listeners.delete(changed);
    };
  };
  loaded(stamp: number): void {
    if (!this.disposed) this.entry.loaded(stamp);
  }
  failed(stamp: number): void {
    if (!this.disposed) this.entry.failed(stamp);
  }
  async retry(): Promise<void> {
    if (!this.disposed) await this.entry.resolve(true);
  }
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    for (const changed of this.subscriptions) this.entry.listeners.delete(changed);
    this.subscriptions.clear();
    this.entry.release();
    this.onRelease();
  }
}
export class ContentReads {
  private readonly entries = new Map<string, ContentReadEntry>();
  private readonly downloads = new Map<string, Promise<void>>();
  private ended = false;
  private leases = 0;
  constructor(
    private readonly view: ContentView,
    private readonly service: AssetService
  ) {
    view.signal.addEventListener("abort", this.dispose, { once: true });
  }
  private listen(): void {
    if (typeof window !== "undefined") {
      window.addEventListener("focus", this.wake);
      window.addEventListener("online", this.wake);
      document.addEventListener("visibilitychange", this.wake);
    }
  }
  private unlisten(): void {
    if (typeof window !== "undefined") {
      window.removeEventListener("focus", this.wake);
      window.removeEventListener("online", this.wake);
      document.removeEventListener("visibilitychange", this.wake);
    }
  }
  private wake = () => {
    if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
    for (const entry of this.entries.values()) entry.wake();
  };
  private entry(id: string): ContentReadEntry {
    if (this.ended || !this.view.current) throw new SessionChangedError();
    let entry = this.entries.get(id);
    if (!entry) {
      const created = new ContentReadEntry(id, this.view, this.service, () => {
        if (this.entries.get(id) === created) this.entries.delete(id);
      });
      entry = created;
      this.entries.set(id, entry);
    }
    return entry;
  }
  read(id: string, signal: AbortSignal): Promise<ContentRead> {
    return this.readWith(id, signal, false);
  }
  retry(id: string, signal: AbortSignal): Promise<ContentRead> {
    return this.readWith(id, signal, true);
  }
  private async readWith(id: string, signal: AbortSignal, force: boolean): Promise<ContentRead> {
    if (signal.aborted) throw new SessionChangedError();
    const entry = this.entry(id);
    entry.readers++;
    let abort!: () => void;
    const aborted = new Promise<never>((_resolve, reject) => {
      abort = () => reject(new SessionChangedError());
      signal.addEventListener("abort", abort, { once: true });
    });
    try {
      return await Promise.race([entry.resolve(force), aborted]);
    } finally {
      signal.removeEventListener("abort", abort);
      entry.releaseReader(signal.aborted);
    }
  }
  invalidate(id: string): void {
    this.entries.get(id)?.invalidate();
  }
  acquire(id: string): ContentReadLease {
    const entry = this.entry(id);
    if (this.leases++ === 0) this.listen();
    const lease = new ReadLease(entry, () => {
      if (--this.leases === 0) this.unlisten();
    });
    void entry.resolve().catch(() => undefined);
    return lease;
  }
  download(id: string): Promise<void> {
    const held = this.downloads.get(id);
    if (held) return held;
    const pending = Promise.resolve()
      .then(async () => {
        const read = await this.read(id, this.view.signal);
        if (!this.view.current) throw new SessionChangedError();
        return downloadAsset(read, this.view.signal);
      })
      .finally(() => {
        if (this.downloads.get(id) === pending) this.downloads.delete(id);
      });
    this.downloads.set(id, pending);
    return pending;
  }
  dispose = (): void => {
    if (this.ended) return;
    this.ended = true;
    for (const entry of this.entries.values()) entry.dispose();
    this.entries.clear();
    this.downloads.clear();
    this.view.signal.removeEventListener("abort", this.dispose);
    this.unlisten();
  };
}
