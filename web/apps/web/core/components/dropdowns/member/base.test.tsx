/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyShown, shown } from "@/lib/fake-controls";
import { MemberDropdownBase } from "./base";

// What the member dropdown keeps for its callers (M4–M6's rows among them) now that a CustomSearchSelect is under it
// (M3 design 7.7, P10 spec 3): it renders on the server with a stand-in for the select (fake-controls.ts), which keeps
// the props it was given, on a desktop unless the test says a phone.

const device = vi.hoisted(() => ({ isMobile: false }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => device }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-me" } }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: () => undefined, workspace: { isUserSuspended: () => false } }),
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));

/** Renders the lead's dropdown, as a caller that defers it or not; gives its markup. */
function render(renderByDefault?: boolean) {
  emptyShown();
  return renderToStaticMarkup(
    <MemberDropdownBase
      getUserDetails={() => undefined}
      memberIds={["u-me"]}
      multiple={false}
      value={null}
      onChange={() => {}}
      buttonVariant="border-with-text"
      placeholder="Lead"
      renderByDefault={renderByDefault}
    />
  );
}

beforeEach(() => {
  device.isMobile = false;
});

describe("MemberDropdownBase", () => {
  it("is its button alone, no select, while its caller defers it", () => {
    const markup = render(false);
    expect([shown.searchSelects.length, markup.includes(">Lead</span></button>")]).toEqual([0, true]);
  });

  it("is a select, its button in the select's, unless its caller defers it", () => {
    const markup = render();
    const button = renderToStaticMarkup(shown.searchSelects[0]?.customButton);
    expect([markup, shown.searchSelects.length, button.includes(">Lead</span></button>")]).toEqual(["", 1, true]);
  });

  it("focuses the search as the list opens on a desktop, not on a phone", () => {
    render();
    const desktop = shown.searchSelects[0]?.focusSearchOnOpen;
    device.isMobile = true;
    render();
    expect([desktop, shown.searchSelects[0]?.focusSearchOnOpen]).toEqual([true, false]);
  });

  it("keeps its list 12 pixels inside the window", () => {
    render();
    expect(shown.searchSelects[0]?.popperModifiers).toEqual([{ name: "preventOverflow", options: { padding: 12 } }]);
  });
});
