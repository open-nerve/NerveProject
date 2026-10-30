/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { compareCodePoints, compareIssueTime } from "@/lib/issue-order";

type ActivityOrder = { id: string; created_at: string; kind?: "activity" | "comment" };
/** Resource-local order is the server's (created_at, id); kind disambiguates a merged display. */
export function compareActivityOrder(a: ActivityOrder, b: ActivityOrder): number {
  return (
    compareIssueTime(a.created_at, b.created_at) ||
    compareCodePoints(a.id, b.id) ||
    compareCodePoints(a.kind ?? "", b.kind ?? "")
  );
}
