/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { projectOf } from "@/store/project/fake-projects";
import { ProjectAuthWrapper } from "./project-wrapper";
import type { ProjectAccess } from "./use-project-fetch";

// What the project wrapper shows for each decision of useProjectFetch (M3 design 3.19, 7.6): the wrapper renders on
// the server with that decision, and stand-ins for the page of an unreachable nerve and for the empty state of the
// join and not-found screens, which keep the props they were given.

type Unavailable = { onRetry: () => void; autoRetry: boolean };
type Screen = { title: string; description: string; actions?: { label: string; onClick: () => void }[] };
const shown = vi.hoisted(() => {
  const access: { current: ProjectAccess } = { current: { kind: "loading" } };
  const unavailable: Unavailable[] = [];
  const screens: Screen[] = [];
  const joinProject = vi.fn((_projectId: string) => Promise.resolve());
  return { access, unavailable, screens, joinProject };
});
vi.mock("./use-project-fetch", () => ({ useProjectFetch: () => shown.access.current }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ joinProject: shown.joinProject }) }));
vi.mock("@/components/account/session-unavailable", () => ({
  SessionUnavailable: (props: Unavailable) => {
    shown.unavailable.push(props);
    return "nerve cannot be reached";
  },
}));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateDetailed: (props: Screen) => {
    shown.screens.push(props);
    return props.title;
  },
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const member = projectOf("WEB", "w-acme");
const seen = projectOf("WEB", "w-acme", { member_role: null });

/** Renders the wrapper of the project WEB, its page a paragraph, with the decision access: gives the markup. */
function render(access: ProjectAccess): string {
  shown.access.current = access;
  return renderToStaticMarkup(
    <ProjectAuthWrapper projectId={member.id}>
      <p>the pages of the project</p>
    </ProjectAuthWrapper>
  );
}

beforeEach(() => {
  shown.unavailable.length = 0;
  shown.screens.length = 0;
  shown.joinProject.mockClear();
});

describe("ProjectAuthWrapper", () => {
  it("shows nothing while nerve's read of the project has not answered", () => {
    expect(render({ kind: "loading" })).toBe("");
    expect([shown.unavailable, shown.screens]).toEqual([[], []]);
  });

  it("shows that nerve cannot be reached, retried when the page asks by the read's retry", () => {
    const retry = vi.fn();
    expect(render({ kind: "unavailable", retry })).toBe("nerve cannot be reached");
    expect(shown.unavailable).toEqual([{ onRetry: retry, autoRetry: false }]);
  });

  it("shows the join screen to one who sees the project and is no member, whose button joins it", () => {
    expect(render({ kind: "not-member", project: seen })).toContain("project_empty_state.no_access.title");
    expect(shown.screens).toMatchObject([
      {
        title: "project_empty_state.no_access.title",
        description: "project_empty_state.no_access.join_description",
        actions: [{ label: "project_empty_state.no_access.cta_primary" }],
      },
    ]);
    shown.screens[0]?.actions?.[0]?.onClick();
    expect(shown.joinProject.mock.calls).toEqual([[member.id]]);
  });

  it("shows that the project is not found, with nothing to do, when it is not found to him", () => {
    expect(render({ kind: "not-found" })).toContain("project_empty_state.invalid_project.title");
    expect(shown.screens).toMatchObject([
      {
        title: "project_empty_state.invalid_project.title",
        description: "project_empty_state.invalid_project.description",
      },
    ]);
    expect(shown.screens[0]?.actions).toBeUndefined();
  });

  it("renders the project's pages to its member", () => {
    expect(render({ kind: "member", project: member })).toBe("<p>the pages of the project</p>");
    expect([shown.unavailable, shown.screens]).toEqual([[], []]);
  });
});
