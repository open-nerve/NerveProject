/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { SwitchAccountModal } from "./switch-account-modal";

// The account switch's sentence (M3 closeout): one key, the address set apart within it, so that each language orders
// the sentence as it does. The stand-in for @nerve/i18n says a key, then the values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@headlessui/react", () => ({ DialogTitle: ({ children }: { children?: ReactNode }) => <h3>{children}</h3> }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { email: "bob@example.com" } }) }));
vi.mock("@/hooks/use-sign-out", () => ({ useSignOut: () => vi.fn() }));

describe("the account switch", () => {
  it("says its sentence as one key, the address set apart within it", () => {
    const markup = renderToStaticMarkup(<SwitchAccountModal isOpen onClose={() => {}} />);
    expect(markup).toContain(
      'onboarding.switch_account.description <span class="text-accent-primary">bob@example.com</span>'
    );
  });
});
