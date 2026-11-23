// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake, Project } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { acme, detailOf, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import type { DescriptionInput } from "@/components/editor/rich-text/description-input";
import { IntakeDetail } from "./detail";

// The description of the selected work item of a project's intake takes images (P5b spec 2.2, R9): the detail gives
// the description the images of its work item (useIssueImages, the issue host), which take uploads while the caller
// may change the work item by the intake's rule and not otherwise; a guest who submitted it among them. The stores,
// the content view and the editor's assets (EditorAssets, whose uploads are fixed as it is built) are real; the
// description is a stand-in that says what images it was given.

const fixture = vi.hoisted(() => ({
  /** The images useIssueImages answered the detail with, and those the description was last drawn with. */
  answered: [] as unknown[],
  drawn: undefined as unknown,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: {},
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/editor/use-issue-images", async (original) => {
  const real = await original<typeof import("@/hooks/editor/use-issue-images")>();
  return {
    useIssueImages: (...args: Parameters<typeof real.useIssueImages>) => {
      const answer = real.useIssueImages(...args);
      fixture.answered.push(answer.images);
      return answer;
    },
  };
});
vi.mock("@/components/editor/rich-text/description-input", () => ({
  DescriptionInput: (props: ComponentProps<typeof DescriptionInput>) => {
    fixture.drawn = props.images;
    const images = props.images ? (props.images.write ? "uploads" : "reads") : "none";
    return <div data-description={String(!!props.disabled)} data-images={images} />;
  },
}));
vi.mock("@/components/issues/title-input", () => ({ IssueTitleInput: () => null }));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("@/components/issues/issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("./issue-properties", () => ({ InboxIssueContentProperties: () => null }));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  fixture.answered = [];
  fixture.drawn = undefined;
  tab = await intakeTab();
  runInAction(() => {
    tab.app.instance.config = {
      product: "Nerve",
      version: "test",
      commit: "test",
      api_version: "v0",
      signup_enabled: true,
      workspace_creation_enabled: true,
      file_size_limit: 10,
    };
  });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const MEMBER = 15 as const;
const GUEST = 5 as const;
const intakeOf = (status: IssueIntake["status"]): IssueIntake => ({
  status,
  snoozed_till: status === "snoozed" ? "2099-01-01T00:00:00Z" : null,
  duplicate_to_id: null,
});

/** The detail of i-x, in p-web's intake as status says and submitted by creator, shown to the caller as role. */
async function shown(role: Project["member_role"], status: IssueIntake["status"], creator: string) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [
    projectOf("WEB", acme.id, { intake_view: true, member_role: role }),
  ]);
  tab.app.issue.issues.acceptRead(
    detailOf("x", { intake: intakeOf(status), created_by_id: creator }),
    tab.app.issue.captureRead()
  );
  await act(async () =>
    root.render(
      <StoreContext.Provider value={tab.app}>
        <IntakeDetail
          workspaceSlug="acme"
          projectId="p-web"
          issueId="i-x"
          isSubmitting="saved"
          setTitleStatus={() => {}}
          setDescriptionStatus={() => {}}
        />
      </StoreContext.Provider>
    )
  );
}
const description = () => host.querySelector<HTMLElement>("[data-description]")!.dataset;

describe("IntakeDetail's description images (P5b spec 2.2)", () => {
  it.each([
    { who: "an admin", role: ADMIN, status: "pending", images: "uploads" },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "snoozed", images: "uploads" },
    { who: "a guest who submitted it", role: GUEST, mine: true, status: "pending", images: "uploads" },
    { who: "a member", role: MEMBER, status: "pending", images: "reads" },
    { who: "an admin", role: ADMIN, status: "declined", images: "reads" },
  ] as const)("gives the description of $who, of a work item $status, its images: $images", async (cell) => {
    await shown(cell.role, cell.status, "mine" in cell ? "u-me" : "u-other");
    // the images are the work item's own, as useIssueImages answered the detail last
    expect(fixture.drawn).toBeDefined();
    expect(fixture.drawn).toBe(fixture.answered.at(-1));
    // which take uploads while the detail lets the caller change the description (by the intake's rule), else read
    expect(description().images).toBe(cell.images);
    expect(description().description).toBe(String(cell.images === "reads"));
  });
});
