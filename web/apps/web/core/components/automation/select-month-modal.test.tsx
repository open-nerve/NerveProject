/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectUpdate } from "@nerve/api-client";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { SelectMonthModal } from "./select-month-modal";

// What the auto-archiving's custom range sends (M3 design 3.19): nerve's ProjectUpdate takes archive_in as a number of
// months and refuses a string. The modal renders on the server with stand-ins for the UI kit's controls
// (fake-controls.ts), which keep the props they were given: the test types the months as the input would, and submits
// the modal's form.

vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme", projectId: "p-web" }) }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));

beforeEach(() => {
  emptyShown();
});

describe("the auto-archiving's custom range", () => {
  it("sends the months typed as a number", async () => {
    const handleChange = vi.fn(async (_formData: Pick<ProjectUpdate, "archive_in">) => {});
    renderToStaticMarkup(
      <SelectMonthModal isOpen initialValues={{ archive_in: 1 }} handleClose={vi.fn()} handleChange={handleChange} />
    );
    const [months] = shown.inputs;
    if (!months) throw new Error("the modal showed no months");
    months.onChange({ target: { value: "6" } });
    await submitModalForm();
    expect(handleChange.mock.calls).toEqual([[{ archive_in: 6 }]]);
  });
});
