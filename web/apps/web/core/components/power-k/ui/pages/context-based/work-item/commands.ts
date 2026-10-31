/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
import { useWorkItemCommandContext } from "./context";
import { useWorkItemCopyCommands } from "./copy-commands";
import { useWorkItemMutationCommands } from "./mutation-commands";

export function usePowerKWorkItemContextBasedCommands(): TPowerKCommandConfig[] {
  const context = useWorkItemCommandContext();
  const mutations = useWorkItemMutationCommands(context);
  const copies = useWorkItemCopyCommands(context);
  return [...mutations, ...copies];
}
