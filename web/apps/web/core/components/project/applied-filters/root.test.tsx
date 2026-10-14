/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ProjectAppliedFiltersList } from "./root";

// The bar of the projects' filters applied (the workspace's projects page): it names each filter in the page's
// language, and each button that removes a filter, or one of its values. The bar renders on the server, with a
// stand-in for the workspace's members (bob) and for the language, which says each key (fake-i18n.ts).

vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: { getWorkspaceMemberDetails: () => ({ member: { display_name: "bob", avatar_url: null } }) },
  }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

describe("the bar of the projects' filters applied", () => {
  it("names each filter, and each button that removes a filter or one of its values", () => {
    const bar = renderToStaticMarkup(
      <ProjectAppliedFiltersList
        appliedFilters={{ access: ["2"], lead: ["u-bob"], members: ["u-bob"], created_at: ["today;custom;custom"] }}
        appliedDisplayFilters={["my_projects"]}
        handleClearAllFilters={() => {}}
        handleRemoveFilter={() => {}}
        handleRemoveDisplayFilter={() => {}}
        filteredProjects={1}
        totalProjects={2}
        alwaysAllowEditing
      />
    );
    const names = [...bar.matchAll(/<span class="text-11 text-tertiary">([^<]*)<\/span>/g)].map(([, name]) => name);
    expect(names).toEqual([
      "workspace_projects.network.label",
      "lead",
      "common.members",
      "workspace_projects.sort.created_at",
      "common.projects",
    ]);
    // the four filters and their four values, and the projects' value: each its button, named; the bar's clearing
    // says what it does
    const buttons = bar.match(/<button[^>]*>/g) ?? [];
    expect(buttons.filter((button) => button.includes('aria-label="common.remove_item"'))).toHaveLength(9);
    expect(buttons).toHaveLength(10);
    expect(bar).toContain("common.clear_all");
  });
});
