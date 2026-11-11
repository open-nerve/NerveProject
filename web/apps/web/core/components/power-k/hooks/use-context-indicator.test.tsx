/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, expect, it, vi } from "vitest";
import { listedCycles } from "@/components/cycles/fake-cycle-lists";
import { web } from "@/store/fake-project-lists";
import { rowOf } from "@/store/planning/fake-cycles";
import { cycleOf } from "@/store/project/fake-projects";
import { useContextIndicator } from "./use-context-indicator";

// The command palette's chip that names what its commands act on (P5a spec 2.7): on a cycle's page, the route's cycle by
// its name, as the cycle store gives it (a stand-in, fake-cycle-lists.ts).

const held = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
}));
vi.mock("react-router", () => ({ useParams: () => held.params }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/use-module", () => ({ useModule: () => ({ getModuleById: () => undefined }) }));

let indicator: string | null;
function Chip() {
  indicator = useContextIndicator({ activeContext: "cycle" });
  return null;
}
const sprint = cycleOf(web, "Sprint", { status: "current" });
beforeEach(() => {
  held.cycles = listedCycles();
  held.cycles.show("cycles", web.id, [rowOf(sprint)]);
});

it("names the route's cycle by its name as the cycle store gives it", () => {
  held.params = { workspaceSlug: "acme", projectId: web.id, cycleId: sprint.id };
  renderToStaticMarkup(<Chip />);
  expect(indicator).toBe("Sprint");
});

it("names nothing of a cycle the store does not give", () => {
  held.params = { workspaceSlug: "acme", projectId: web.id, cycleId: "c-gone" };
  renderToStaticMarkup(<Chip />);
  expect(indicator).toBeNull();
});
