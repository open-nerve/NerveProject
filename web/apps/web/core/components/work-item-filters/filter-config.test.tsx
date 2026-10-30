// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { WorkItemFilterStore, type IWorkItemFilterInstance } from "@nerve/shared-state";
import { COLLECTION_OPERATOR, EIssuesStoreType, FILTER_FIELD_TYPE, type TWorkItemFilterExpression } from "@nerve/types";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { toFilterExpression, toMemberFilters, toProjectFilters } from "@/lib/issue-filters";
import { FilterItem } from "@/components/rich-filters/filter-item/root";
import { WorkItemFiltersHOC } from "./filters-hoc/base";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "filters" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/work-item-filters/use-work-item-filters", () => ({ useWorkItemFilters: () => filters }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => tab.app.memberRoot }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let filters: WorkItemFilterStore;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let changes: TWorkItemFilterExpression[];
const projectID = "11111111-1111-4111-8111-111111111111";
const Chips = observer(function Chips({ filter }: { filter: IWorkItemFilterInstance }) {
  return filter.allConditions.map((condition) => (
    <FilterItem key={condition.id} condition={condition} filter={filter} showTransition={false} />
  ));
});
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  filters = new WorkItemFilterStore();
  changes = [];
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
async function render(
  page: "issues" | "profile_issues" | "archived_issues",
  expression: TWorkItemFilterExpression = {}
) {
  await act(async () =>
    renderer.render(
      <WorkItemFiltersHOC
        entityId="scope"
        entityType={EIssuesStoreType.PROJECT}
        workspaceSlug="ws"
        memberIds={[]}
        stateIds={[]}
        labelIds={[]}
        projectIds={[]}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE[page]!.filters}
        initialWorkItemFilters={{
          richFilters: expression,
          displayFilters: undefined,
          displayProperties: undefined,
          kanbanFilters: undefined,
        }}
        updateFilters={(value) => changes.push(value)}
      >
        {({ filter }) => filter && <Chips filter={filter} />}
      </WorkItemFiltersHOC>
    )
  );
  return filters.getFilter(EIssuesStoreType.PROJECT, "scope")!;
}

it("renders a saved subscriber as an editable native filter and preserves it when priority changes", async () => {
  const filter = await render("issues", toFilterExpression({ subscriber_id: ["me"], priority: ["high"] }));
  expect(host.textContent).not.toContain("Invalid");
  expect(host.textContent).toContain("common.filter_options.subscribers");
  const priority = filter.allConditions.find((condition) => condition.property === "priority")!;
  await act(async () => filter.updateConditionValue(priority.id, ["low"]));
  expect(toProjectFilters(changes.at(-1)!)).toEqual({ subscriber_id: ["me"], priority: ["low"] });
});

it("offers only each page's native conditions, including project restrictions on a profile", async () => {
  const project = await render("issues");
  expect(project.configManager.allAvailableConfigs.map((config) => config.id).toSorted()).toEqual([
    "assignee_id",
    "created_by_id",
    "label_id",
    "mention_id",
    "priority",
    "start_date",
    "state_group",
    "state_id",
    "subscriber_id",
    "target_date",
  ]);
  await act(async () => renderer.render(null));
  const profile = await render("profile_issues", { and: [{ project_id__in: projectID }, { priority__in: "high" }] });
  expect(profile.configManager.getConfigByProperty("project_id")?.isEnabled).toBe(true);
  const priority = profile.allConditions.find((condition) => condition.property === "priority")!;
  await act(async () => profile.updateConditionValue(priority.id, ["low"]));
  expect(toMemberFilters(changes.at(-1)!)).toEqual({ project_id: [projectID], priority: ["low"] });
});

it("offers me for user filters and none only for assignees and labels, even with an empty directory", async () => {
  const filter = await render("issues");
  await Promise.all(
    (
      [
        ["assignee_id", ["me", "none"]],
        ["created_by_id", ["me"]],
        ["mention_id", ["me"]],
        ["subscriber_id", ["me"]],
        ["label_id", ["none"]],
      ] as const
    ).map(async ([property, expected]) => {
      const config = filter.configManager.getConfigByProperty(property)?.getOperatorConfig(COLLECTION_OPERATOR.IN);
      expect(config?.type).toBe(FILTER_FIELD_TYPE.MULTI_SELECT);
      if (config?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error(`Missing multi-select for ${property}`);
      const options = typeof config.getOptions === "function" ? await config.getOptions() : config.getOptions;
      expect(options.map((option) => option.value).toSorted()).toEqual([...expected].toSorted());
    })
  );
});

it("keeps the archived picker within its allowed native filters", async () => {
  const filter = await render("archived_issues");
  expect(filter.configManager.allAvailableConfigs.map((config) => config.id).toSorted()).toEqual([
    "assignee_id",
    "created_by_id",
    "label_id",
    "priority",
    "start_date",
    "state_group",
    "state_id",
    "target_date",
  ]);
});
