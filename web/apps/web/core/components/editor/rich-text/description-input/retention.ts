/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useMemo } from "react";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import type { AutosaveRetention } from "../../autosave-retention";
export type DescriptionValue = { id: string; description_html: string; isMigrationUpdate: boolean };
/** Retain text, not a prepared write or the old migration-only exemption. */
export function useDescriptionRetention(id: string): AutosaveRetention<DescriptionValue> | undefined {
  const slot = useIssueDetailInput()?.slot("description");
  return useMemo(
    () =>
      slot
        ? {
            read: () => {
              const kept = slot.read();
              return (
                kept && {
                  ...kept,
                  value: { id, description_html: kept.value, isMigrationUpdate: false },
                  confirmed: { id, description_html: kept.confirmed, isMigrationUpdate: false },
                }
              );
            },
            store: (snapshot) =>
              slot.store({
                ...snapshot,
                value: snapshot.value.description_html,
                confirmed: snapshot.confirmed.description_html,
              }),
            get canFlushOnUnmount() {
              return slot.canFlushOnUnmount;
            },
          }
        : undefined,
    [id, slot]
  );
}
