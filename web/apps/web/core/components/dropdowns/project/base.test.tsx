// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { ProjectDropdownBase } from "./base";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// Preserve the original caller-supplied tabIndex contract, now on the single native trigger.
it("puts its only button where the caller's tabIndex says and never makes the root a Tab stop", async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () =>
      root.render(
        <ProjectDropdownBase
          buttonVariant="border-with-text"
          getProjectById={() => undefined}
          projectIds={[]}
          multiple={false}
          value={null}
          onChange={() => undefined}
          tabIndex={0}
        />
      )
    );
    expect(host.querySelectorAll("button")).toHaveLength(1);
    expect(host.querySelector("button")?.tabIndex).toBe(0);
    expect(host.querySelector("div[tabindex]")).toBeNull();
  } finally {
    await act(async () => root.unmount());
    host.remove();
  }
});
