// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { expect, it } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { draftOf } from "@/store/issue/fake-issues";
import {
  click,
  close,
  dirty,
  html,
  host,
  mountExisting,
  original,
  submit,
  type,
  useFormFixture,
  writes,
} from "./fake-form";

useFormFixture();
it.each(["issue", "draft"] as const)("migrationDoesNotDirtyExisting %s or start a business write", async (kind) => {
  await mountExisting(kind);
  expect(dirty()).toBe("false");
  await click("migrate description");
  expect(html()).toBe('<p data-id="migrated">Original</p>');
  expect(dirty()).toBe("false");
  expect(writes()).toEqual([]);
  await click("discard");
  expect(close).toHaveBeenCalledOnce();
  expect(host.textContent).not.toContain("save_this_draft");
  expect(writes()).toEqual([]);
});
it.each(["issue", "draft"] as const)("migrationPreservesRealEdits in an existing %s", async (kind) => {
  await mountExisting(kind);
  await type('textarea[aria-label="description"]', "<p>User edit</p>");
  expect(dirty()).toBe("true");
  await click("migrate description");
  expect(html()).toBe('<p data-id="migrated">User edit</p>');
  expect(dirty()).toBe("true");
  await submit();
  await act(async () => until(() => writes().length === 1, "explicit save"));
  const request = writes()[0]!;
  if (kind === "issue") {
    expect(request.body).toEqual({ description_html: '<p data-id="migrated">User edit</p>' });
    await act(async () =>
      request.answer(json(200, { ...original(), description_html: '<p data-id="migrated">User edit</p>' }))
    );
  } else {
    expect(request.body).toMatchObject({ payload: { description_html: '<p data-id="migrated">User edit</p>' } });
    expect(request.body).not.toHaveProperty("skip_activity");
    await act(async () =>
      request.answer(
        json(200, {
          ...draftOf("d"),
          payload: { name: "Existing", description_html: '<p data-id="migrated">User edit</p>' },
        })
      )
    );
  }
});
it("a title edit survives a description migration and remains a normal audited update", async () => {
  await mountExisting("issue");
  await type("#name", "Renamed");
  await click("migrate description");
  expect(dirty()).toBe("true");
  await submit();
  await act(async () => until(() => writes().length === 1, "title and migrated description"));
  expect(writes()[0]!.body).toEqual({ name: "Renamed", description_html: '<p data-id="migrated">Original</p>' });
  await act(async () =>
    writes()[0]!.answer(
      json(200, { ...original(), name: "Renamed", description_html: '<p data-id="migrated">Original</p>' })
    )
  );
});
it("an explicit save of only an existing issue's migrated HTML uses boolean skip_activity", async () => {
  await mountExisting("issue");
  await click("migrate description");
  await submit();
  await act(async () => until(() => writes().length === 1, "explicit migration save"));
  expect(writes()[0]!.body).toEqual({ description_html: '<p data-id="migrated">Original</p>', skip_activity: true });
  await act(async () =>
    writes()[0]!.answer(json(200, { ...original(), description_html: '<p data-id="migrated">Original</p>' }))
  );
});
