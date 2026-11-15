/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueUpdate } from "@nerve/api-client";
import { ISSUE_PRIORITIES } from "@nerve/constants";
import type { IssueChange } from "@nerve/types";
import { Signal } from "lucide-react";
import {
  CyclesOutline,
  DeleteOutline,
  LabelsOutline,
  MembersOutline,
  StateOutline,
  UserMinusOutline,
  UserPlusOutline,
} from "@makeplane/propel/icons";
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { sessionGuard } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { planningOn } from "@/lib/planning-switch";
import { changeBody } from "@/store/issue/issue-write-fields";
import type { WorkItemCommandContext } from "./context";

const toggled = (ids: string[], id: string) => (ids.includes(id) ? ids.filter((held) => held !== id) : [...ids, id]);

export function useWorkItemMutationCommands({ root, read, capture }: WorkItemCommandContext): TPowerKCommandConfig[] {
  const { toggleDeleteIssueModal } = useCommandPalette();
  const refuse = useRefusalToast();
  const editable = () => !!read()?.canEdit;
  const deletable = () => !!read()?.canDelete;
  const assigned = !!read()?.issue.assignee_ids.includes(root.rootStore.user.data?.id ?? "");
  const update = async (makeBody: (held: Issue) => IssueUpdate) => {
    const target = read();
    if (!target?.canEdit) return;
    const current = capture();
    if (!current()) return;
    const inSession = sessionGuard();
    try {
      await root.writes.updateIssue(
        target.issue.id,
        (held) => {
          // Read the authoritative value when this command reaches the existing issue queue.
          if (!current() || !inSession() || !read()?.canEdit) throw new SessionChangedError();
          return makeBody(held);
        },
        undefined,
        root.scope(target.workspaceSlug, target.project.id)
      );
    } catch (error) {
      if (current() && inSession() && !(error instanceof SessionChangedError)) refuse(error);
    }
  };
  /**
   * A change of the work item as its dropdowns send one (IssueChange, P7a spec 2.2, R3), through the guards above: its
   * request made in the work item's turn (changeBody), its modules' additions and removals applied to the set it then
   * holds; the type takes no whole set (P7c spec 2.5).
   */
  const change = (requested: IssueChange) => update((held) => changeBody(requested, held.module_ids));
  /** The work item may be changed and its project's cycles are on (P7c spec 2.5). */
  const cyclesEditable = () => {
    const target = read();
    return !!target?.canEdit && planningOn(target.project, "cycles");
  };
  const assign = (value: unknown) =>
    typeof value === "string" && value
      ? update((held) => ({ assignee_ids: toggled(held.assignee_ids, value) }))
      : undefined;
  return [
    {
      id: "change_work_item_state",
      i18n_title: "power_k.contextual_actions.work_item.change_state",
      icon: StateOutline,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-state",
      shortcut: "s",
      closeOnSelect: true,
      isEnabled: editable,
      isVisible: editable,
      onSelect: (value) => (typeof value === "string" && value ? update(() => ({ state_id: value })) : undefined),
    },
    {
      id: "change_work_item_priority",
      i18n_title: "power_k.contextual_actions.work_item.change_priority",
      icon: Signal,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-priority",
      shortcut: "p",
      closeOnSelect: true,
      isEnabled: editable,
      isVisible: editable,
      onSelect: (value) => {
        const priority = ISSUE_PRIORITIES.find((item) => item.key === value);
        if (priority) return update(() => ({ priority: priority.key }));
      },
    },
    {
      id: "change_work_item_assignees",
      i18n_title: "power_k.contextual_actions.work_item.change_assignees",
      icon: MembersOutline,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-assignee",
      shortcut: "a",
      closeOnSelect: false,
      isEnabled: editable,
      isVisible: editable,
      onSelect: assign,
    },
    {
      id: "assign_work_item_to_me",
      i18n_title: assigned
        ? "power_k.contextual_actions.work_item.unassign_from_me"
        : "power_k.contextual_actions.work_item.assign_to_me",
      icon: assigned ? UserMinusOutline : UserPlusOutline,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      shortcut: "i",
      closeOnSelect: true,
      isEnabled: editable,
      isVisible: editable,
      action: () => assign(root.rootStore.user.data?.id),
    },
    {
      // P7c spec 2.5: its page offers No cycle (null) and the cycles it may join; the cycle it is in sends nothing.
      id: "add_work_item_to_cycle",
      i18n_title: "power_k.contextual_actions.work_item.add_to_cycle",
      icon: CyclesOutline,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-cycle",
      modifierShortcut: "shift+c",
      closeOnSelect: true,
      isEnabled: cyclesEditable,
      isVisible: cyclesEditable,
      onSelect: (value) => {
        const cycleId = value === null ? null : typeof value === "string" && value ? value : undefined;
        if (cycleId === undefined || cycleId === read()?.issue.cycle_id) return;
        return change({ cycle_id: cycleId });
      },
    },
    {
      id: "add_work_item_labels",
      i18n_title: "power_k.contextual_actions.work_item.add_labels",
      icon: LabelsOutline,
      group: "contextual",
      contextType: "work-item",
      type: "change-page",
      page: "update-work-item-labels",
      shortcut: "l",
      closeOnSelect: false,
      isEnabled: editable,
      isVisible: editable,
      onSelect: (value) => {
        if (
          typeof value !== "object" ||
          value === null ||
          !("id" in value) ||
          typeof value.id !== "string" ||
          !value.id
        )
          return;
        const id = value.id;
        return update((held) => ({ label_ids: toggled(held.label_ids, id) }));
      },
    },
    {
      id: "delete_work_item",
      i18n_title: "power_k.contextual_actions.work_item.delete",
      icon: DeleteOutline,
      group: "contextual",
      contextType: "work-item",
      type: "action",
      modifierShortcut: "cmd+backspace",
      closeOnSelect: true,
      isEnabled: deletable,
      isVisible: deletable,
      action: () => {
        if (capture()() && deletable()) toggleDeleteIssueModal(true);
      },
    },
  ];
}
