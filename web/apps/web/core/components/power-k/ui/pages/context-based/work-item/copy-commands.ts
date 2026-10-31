/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { TicketCheck } from "lucide-react";
import { LinkOutline, TextOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
import { useCopyLink, useCopyText } from "@/hooks/use-copy-link";
import type { WorkItemCommandContext } from "./context";

export function useWorkItemCopyCommands({ read, capture }: WorkItemCommandContext): TPowerKCommandConfig[] {
  const copyText = useCopyText();
  const copyLink = useCopyLink();
  const { t } = useTranslation();
  const available = () => !!read();
  return [
    {
      id: "copy_work_item_id",
      i18n_title: "power_k.contextual_actions.work_item.copy_id",
      icon: TicketCheck,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      modifierShortcut: "cmd+.",
      closeOnSelect: true,
      isEnabled: available,
      isVisible: available,
      action: async () => {
        if (!capture()()) return;
        const current = read();
        if (current)
          await copyText(
            `${current.project.identifier}-${current.issue.sequence_id}`,
            t("power_k.contextual_actions.work_item.copy_id_toast_success")
          );
      },
    },
    {
      id: "copy_work_item_title",
      i18n_title: "power_k.contextual_actions.work_item.copy_title",
      icon: TextOutline,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      modifierShortcut: "cmd+shift+'",
      closeOnSelect: true,
      isEnabled: available,
      isVisible: available,
      action: async () => {
        if (!capture()()) return;
        const current = read();
        if (current)
          await copyText(current.issue.name, t("power_k.contextual_actions.work_item.copy_title_toast_success"));
      },
    },
    {
      id: "copy_work_item_url",
      i18n_title: "power_k.contextual_actions.work_item.copy_url",
      icon: LinkOutline,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      modifierShortcut: "cmd+shift+,",
      closeOnSelect: true,
      isEnabled: available,
      isVisible: available,
      action: async () => {
        if (!capture()()) return;
        const current = read();
        if (current)
          await copyLink(
            `/${current.workspaceSlug}/browse/${current.project.identifier}-${current.issue.sequence_id}`,
            t("power_k.contextual_actions.work_item.copy_url_toast_success")
          );
      },
    },
  ];
}
