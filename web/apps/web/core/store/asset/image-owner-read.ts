/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
/** The image consumer observes real owner reads; it never owns their transport or DTOs. */
export type ImageOwner =
  | { kind: "self" }
  | { kind: "workspace" }
  | { kind: "project"; id: string }
  | { kind: "member"; id: string };
export type ObserveProfileRead = <T>(owner: ImageOwner, read: () => Promise<T>) => Promise<T>;
export const imageOwnerKey = (owner: ImageOwner): string =>
  owner.kind === "self"
    ? "self"
    : owner.kind === "workspace"
      ? "workspace-directory"
      : JSON.stringify([owner.kind, owner.id]);
