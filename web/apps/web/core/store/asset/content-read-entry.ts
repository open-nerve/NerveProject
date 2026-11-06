/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Asset, Content } from "@nerve/api-client";
import { capabilityFailure } from "@/lib/asset-capability";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { AssetService } from "@/services/asset.service";
import type { ContentView } from "./content-view";
import { matchesContent } from "./content-target";
export type ContentRead = { asset: Asset; content: Content };
export type ContentSnapshot =
  | { status: "loading" | "unavailable"; stamp: number; error?: unknown }
  | ({ status: "ready"; stamp: number; error?: undefined } & ContentRead);
/** One view and asset share metadata, capability, expiry and the automatic error budget. */
export class ContentReadEntry {
  snapshot: ContentSnapshot = { status: "loading", stamp: 0 };
  readonly listeners = new Set<() => void>();
  leases = 0;
  readers = 0;
  private version = 0;
  private ended = false;
  private automaticUsed = false;
  /** The last renewal at expiry never reached a server, e.g. it ran on waking before the network was back. */
  private unsent = false;
  private pending?: Promise<ContentRead>;
  private controller?: AbortController;
  private timer?: ReturnType<typeof setTimeout>;
  constructor(
    readonly id: string,
    private readonly view: ContentView,
    private readonly service: AssetService,
    private readonly forget: () => void
  ) {}
  get current(): boolean {
    return !this.ended && this.view.current;
  }
  private publish(snapshot: ContentSnapshot): void {
    this.snapshot = snapshot;
    for (const changed of this.listeners) changed();
  }
  private stopTimer(): void {
    if (this.timer !== undefined) clearTimeout(this.timer);
    this.timer = undefined;
  }
  private arm(): void {
    this.stopTimer();
    if (!this.current || !this.leases || this.snapshot.status !== "ready") return;
    const delay = Date.parse(this.snapshot.content.expires_at) - Date.now();
    this.timer = setTimeout(
      () => {
        this.timer = undefined;
        this.wake();
      },
      Math.max(1, Math.min(delay, 2_147_483_647))
    );
  }
  wake(): void {
    if (!this.current || !this.leases) return;
    if (this.snapshot.status === "ready") {
      if (Date.parse(this.snapshot.content.expires_at) > Date.now()) {
        this.arm();
        return;
      }
    } else if (!this.unsent) return;
    this.renew();
  }
  private renew(): void {
    this.unsent = false;
    this.resolve(true).catch((error: unknown) => {
      // A server's answer waits for an explicit retry; a request the network never carried waits for the next wake.
      if (this.current && error instanceof TypeError) this.unsent = true;
    });
  }
  resolve(force = false): Promise<ContentRead> {
    if (!this.current) return Promise.reject(new SessionChangedError());
    if (this.pending) return this.pending;
    if (!force && this.snapshot.status === "ready" && Date.parse(this.snapshot.content.expires_at) > Date.now())
      return Promise.resolve({ asset: this.snapshot.asset, content: this.snapshot.content });
    const version = ++this.version;
    const controller = new AbortController();
    this.controller = controller;
    this.stopTimer();
    const current = () => this.current && version === this.version;
    const pending = Promise.resolve()
      .then(async () => {
        if (!current()) throw new SessionChangedError();
        const asset = await this.service.read(this.id, controller.signal);
        if (!current()) throw new SessionChangedError();
        if (asset.id !== this.id || asset.state !== "uploaded" || !matchesContent(asset, this.view.host))
          throw new Error("Asset does not belong to this content view");
        const content = await this.service.content(this.id, controller.signal);
        if (!current()) throw new SessionChangedError();
        if (capabilityFailure(content.url, content.expires_at))
          throw new Error("Content authorization is invalid or expired");
        // A synchronous image event may request a new authorization from this notification.
        this.pending = undefined;
        this.controller = undefined;
        this.unsent = false;
        this.publish({ status: "ready", stamp: version, asset, content });
        if (!current()) throw new SessionChangedError();
        this.arm();
        return { asset, content };
      })
      .catch((error: unknown) => {
        if (current()) this.publish({ status: "unavailable", stamp: version, error });
        throw error;
      })
      .finally(() => {
        if (this.pending === pending) {
          this.pending = undefined;
          this.controller = undefined;
        }
      });
    this.pending = pending;
    this.publish({ status: "loading", stamp: version });
    return pending;
  }
  loaded(stamp: number): void {
    if (this.current && this.snapshot.status === "ready" && this.snapshot.stamp === stamp) this.automaticUsed = false;
  }
  failed(stamp: number): void {
    if (!this.current || this.snapshot.status !== "ready" || this.snapshot.stamp !== stamp) return;
    const retry = !this.automaticUsed;
    this.automaticUsed = true;
    this.stopTimer();
    this.publish({ status: "unavailable", stamp, error: new Error("Image could not be loaded") });
    if (this.current && retry) void this.resolve(true).catch(() => undefined);
  }
  retain(): void {
    this.leases++;
    this.arm();
  }
  release(): void {
    this.leases--;
    if (!this.leases && !this.readers) this.dispose();
  }
  releaseReader(cancelled: boolean): void {
    this.readers--;
    if (!this.readers && !this.leases && cancelled) this.dispose();
  }
  dispose(): void {
    if (this.ended) return;
    this.ended = true;
    ++this.version;
    this.controller?.abort();
    this.controller = undefined;
    this.pending = undefined;
    this.stopTimer();
    this.publish({ status: "unavailable", stamp: this.version });
    this.listeners.clear();
    this.forget();
  }
}
