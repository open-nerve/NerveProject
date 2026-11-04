/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Asset, Content } from "@nerve/api-client";
import { capabilityFailure } from "./asset-capability";

type DownloadFailureKind =
  | "invalid-grant"
  | "expired"
  | "invalid-size"
  | "size-mismatch"
  | "cancelled"
  | "timeout"
  | "http"
  | "network"
  | "download";
export class DownloadFailure extends Error {
  constructor(
    readonly kind: DownloadFailureKind,
    readonly status?: number
  ) {
    super(`Download ${kind}`);
    this.name = "DownloadFailure";
  }
}

/** This byte transport has no API middleware and never persists the capability URL. */
export async function downloadAsset(
  { asset, content }: { asset: Asset; content: Content },
  signal: AbortSignal
): Promise<void> {
  if (signal.aborted) throw new DownloadFailure("cancelled");
  const invalid = capabilityFailure(content.url, content.expires_at);
  if (invalid) throw new DownloadFailure(invalid);
  if (!Number.isSafeInteger(asset.size) || asset.size < 1 || asset.size > 104857600)
    throw new DownloadFailure("invalid-size");
  const controller = new AbortController();
  return new Promise<void>((resolve, reject) => {
    let ended = false;
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
    let link: HTMLAnchorElement | undefined;
    let objectURL: string | undefined;
    const finish = (error?: DownloadFailure) => {
      if (ended) return;
      ended = true;
      clearTimeout(timer);
      signal.removeEventListener("abort", onAbort);
      if (error) controller.abort();
      const active = reader;
      reader = undefined;
      if (active) void releaseReader(active);
      link?.remove();
      if (objectURL) URL.revokeObjectURL(objectURL);
      if (error) reject(error);
      else resolve();
    };
    const onAbort = () => finish(new DownloadFailure("cancelled"));
    const timer = setTimeout(() => finish(new DownloadFailure("timeout")), 15 * 60_000);
    signal.addEventListener("abort", onAbort, { once: true });
    const read = async () => {
      const response = await fetch(content.url, {
        method: "GET",
        signal: controller.signal,
        credentials: "omit",
        redirect: "error",
        referrerPolicy: "no-referrer",
      });
      if (ended || !response.ok) {
        void response.body?.cancel().catch(() => undefined);
        if (!ended) finish(new DownloadFailure("http", response.status));
        return;
      }
      if (!response.body) throw new DownloadFailure("size-mismatch");
      reader = response.body.getReader();
      const chunks: BlobPart[] = [];
      let size = 0;
      while (true) {
        // oxlint-disable-next-line no-await-in-loop -- Serial consumption enforces the byte limit before reading another chunk.
        const { done, value } = await reader.read();
        if (ended) return;
        if (done) break;
        size += value.byteLength;
        if (size > asset.size) throw new DownloadFailure("size-mismatch");
        chunks.push(value.slice());
      }
      if (size !== asset.size) throw new DownloadFailure("size-mismatch");
      try {
        objectURL = URL.createObjectURL(new Blob(chunks, { type: asset.content_type }));
        link = document.createElement("a");
        link.href = objectURL;
        link.download = asset.name;
        document.body.appendChild(link);
        link.click();
      } catch {
        throw new DownloadFailure("download");
      }
      finish();
    };
    void read().catch((error: unknown) =>
      finish(error instanceof DownloadFailure ? error : new DownloadFailure("network"))
    );
  });
}

async function releaseReader(reader: ReadableStreamDefaultReader<Uint8Array>): Promise<void> {
  try {
    await reader.cancel();
  } catch {
    // Cancellation of an already failed stream must not replace the original terminal result.
  } finally {
    reader.releaseLock();
  }
}
