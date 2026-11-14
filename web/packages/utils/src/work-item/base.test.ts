/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { expect, it } from "vitest";
import { getComputedDisplayProperties } from "./base";

// A list's display properties on a page that keeps them in the browser (M6 P7a spec 2.5): a work item's cycle and
// modules show unless the page's settings hide them, as nerve's own defaults show them (DefaultListDisplayProperties).

it("shows a work item's cycle and modules unless the settings say otherwise", () => {
  expect(getComputedDisplayProperties({})).toMatchObject({ cycle: true, modules: true });
  expect(getComputedDisplayProperties()).toMatchObject({ cycle: true, modules: true });
});

it("keeps a cycle or modules the settings hide hidden", () => {
  expect(getComputedDisplayProperties({ cycle: false })).toMatchObject({ cycle: false, modules: true });
  expect(getComputedDisplayProperties({ modules: false })).toMatchObject({ cycle: true, modules: false });
});
