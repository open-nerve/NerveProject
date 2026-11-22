/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { SubscribeOutline, UnsubscribeOutline } from "@makeplane/propel/icons";
import { useSubscriptionToggle } from "@/components/issues/issue-detail/use-subscription-toggle";
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
import type { WorkItemCommandContext } from "./context";

/**
 * The work item's command that subscribes the caller to it or unsubscribes him (P4 spec 2.6, R13, R14), by Shift+S:
 * offered where its detail's button is (the subscription store's subscribable, one gate for both), titled by its
 * detail's is_subscribed, out of use while a change of it is pending. It sends the change as the button does
 * (useSubscriptionToggle), the palette's scope captured as it runs: a command kept from another work item, or from
 * before the scope changed, sends nothing.
 */
export function useWorkItemSubscriptionCommand({ root, read, capture }: WorkItemCommandContext): TPowerKCommandConfig {
  const toggle = useSubscriptionToggle();
  const { subscription } = root.issueDetail;
  /** The work item's detail while the command is offered. */
  const offered = () => {
    const target = read();
    return target && subscription.subscribable(target.workspaceSlug, target.project.id, target.issue.id);
  };
  const subscribed = !!offered()?.is_subscribed;
  return {
    id: "subscribe_work_item",
    i18n_title: subscribed
      ? "power_k.contextual_actions.work_item.unsubscribe"
      : "power_k.contextual_actions.work_item.subscribe",
    icon: subscribed ? UnsubscribeOutline : SubscribeOutline,
    group: "contextual",
    contextType: "work-item",
    type: "action",
    modifierShortcut: "shift+s",
    closeOnSelect: true,
    isEnabled: () => {
      const detail = offered();
      return !!detail && !subscription.isPending(detail.id);
    },
    isVisible: () => !!offered(),
    action: async () => {
      const target = read();
      if (!capture()() || !target) return;
      await toggle(target.workspaceSlug, target.project.id, target.issue.id);
    },
  };
}
