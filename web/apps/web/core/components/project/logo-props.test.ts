/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { LogoProps } from "@nerve/api-client";
import type { TChangeHandlerProps } from "@nerve/propel/emoji-icon-picker";
import { logoPropsOf } from "./logo-props";

// A project's icon as the picker gives it and as nerve keeps it (M3 design 3.19): LogoProps's closed shape.

describe("logoPropsOf", () => {
  it.each<{ picked: TChangeHandlerProps; kept: LogoProps }>([
    {
      picked: { type: "emoji", value: "128640" },
      kept: { in_use: "emoji", emoji: { value: "128640" } },
    },
    {
      picked: { type: "icon", value: { name: "rocket", color: "#5e6ad2" } },
      kept: { in_use: "icon", icon: { name: "rocket", color: "#5e6ad2" } },
    },
  ])("keeps an $picked.type picked as nerve keeps it", ({ picked, kept }) => {
    expect(logoPropsOf(picked)).toEqual(kept);
  });
});
