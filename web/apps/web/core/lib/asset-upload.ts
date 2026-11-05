/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { PreparedAsset } from "@nerve/api-client";
import { capabilityFailure } from "./asset-capability";

export type UploadFailureKind = "network" | "http" | "timeout" | "cancelled" | "expired" | "invalid-grant";
export class UploadFailure extends Error {
  constructor(
    readonly kind: UploadFailureKind,
    readonly status?: number
  ) {
    super(`Upload ${kind}`);
    this.name = "UploadFailure";
  }
}

/** No API middleware, redirects or ambient credentials may cross this storage boundary. */
export async function putAsset(file: File, grant: PreparedAsset["upload"], signal: AbortSignal): Promise<void> {
  if (signal.aborted) throw new UploadFailure("cancelled");
  const headers = checkedHeaders(grant);
  const controller = new AbortController();
  return new Promise<void>((resolve, reject) => {
    let ended = false;
    const cleanup = () => {
      clearTimeout(timer);
      signal.removeEventListener("abort", onAbort);
    };
    const finish = (error?: UploadFailure) => {
      if (ended) return;
      ended = true;
      cleanup();
      if (error) reject(error);
      else resolve();
    };
    const abort = (kind: "cancelled" | "timeout") => {
      if (ended) return;
      finish(new UploadFailure(kind));
      controller.abort();
    };
    const onAbort = () => abort("cancelled");
    const timer = setTimeout(() => abort("timeout"), 15 * 60_000);
    signal.addEventListener("abort", onAbort, { once: true });
    try {
      // Validation must never normalize the original URL: even equivalent encodings may invalidate a signature.
      void fetch(grant.url, {
        method: "PUT",
        body: file,
        headers,
        signal: controller.signal,
        credentials: "omit",
        redirect: "error",
        referrerPolicy: "no-referrer",
      }).then(
        (response) => {
          // This response has no application data. Do not read XML into an error or wait on a slow response body.
          void response.body?.cancel().catch(() => undefined);
          return finish(response.ok ? undefined : new UploadFailure("http", response.status));
        },
        () => finish(new UploadFailure("network"))
      );
    } catch {
      finish(new UploadFailure("network"));
    }
  });
}

function checkedHeaders(grant: PreparedAsset["upload"]): Headers {
  const invalid = () => new UploadFailure("invalid-grant");
  const { url, method, headers, expires_at: expiresAt } = grant;
  if (method !== "PUT") throw invalid();
  const header = headers[0];
  if (
    headers.length !== 1 ||
    header?.name.toLowerCase() !== "content-type" ||
    !header.value.trim() ||
    hasControl(header.value)
  )
    throw invalid();
  const failure = capabilityFailure(url, expiresAt);
  if (failure) throw new UploadFailure(failure);
  try {
    return new Headers([[header.name, header.value]]);
  } catch {
    throw invalid();
  }
}

function hasControl(value: string): boolean {
  return [...value].some((char) => char.charCodeAt(0) < 32 || char.charCodeAt(0) === 127);
}
