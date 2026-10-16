/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { ProjectFeatureUpdate } from "./project-feature-update";

// The sentence the features step of a project's creation ends with (M3 closeout, the fix wave's A14): one key with the
// project in it, so that each language orders it as it does. The step renders on the server, its features' list left
// out, with a stand-in for @nerve/i18n whose text is its key, then the values it was given (fake-i18n-values.ts).

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/components/project/settings/features-list", () => ({ ProjectFeaturesList: () => null }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => ({ name: "Web", logo_props: { in_use: "emoji", emoji: { value: "" } } }),
  }),
}));

describe("the features step of a project's creation", () => {
  it("says the project is created in one sentence, the project's name and logo within it", () => {
    const step = renderToStaticMarkup(
      <MemoryRouter>
        <ProjectFeatureUpdate workspaceSlug="acme" projectId="web" onClose={() => {}} />
      </MemoryRouter>
    );
    expect(step).toMatch(/<p [^>]*>project_dialogs\.project_created <span [^>]*>.*Web<\/span><\/p>/);
  });
});
