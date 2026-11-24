/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A view's page's draft of the view (P6a spec 2.6, R2; re-96 I2): seeded by one read of the view, then changed only in
// the page, a field at a time, never sent; replaced by nerve's answer to "Update view" only while it lasts; ended by
// the page, with a seed then out. The read is the view store's (ViewStore.fetchView), which the test answers itself.
import { describe, expect, it } from "vitest";
import type { View } from "@nerve/api-client";
import { track } from "@/lib/auth/fake-time";
import { projectOf } from "@/store/project/fake-projects";
import { viewOf } from "@/store/views/fake-views";
import { ViewDraftService } from "./view-draft.service";

const web = projectOf("WEB", "id-ws");
/** The view as nerve holds it: urgent work items. */
const saved = viewOf(web, "Urgent", "u-me", { filters: { priority: ["urgent"] } });
const V = saved.id;
const fieldsOf = (view: View) => ({
  filters: view.filters,
  display_filters: view.display_filters,
  display_properties: view.display_properties,
});
/** nerve's answer to "Update view": the view with other filters and another layout. */
const changed: View = {
  ...saved,
  filters: { priority: ["high"] },
  display_filters: { ...saved.display_filters, layout: "kanban", group_by: "priority" },
};

/** Lets the promises settled meanwhile run on. */
const settled = () => new Promise((resolve) => setTimeout(resolve, 0));
/** A draft service over reads of the view the test answers, one at a time. */
function drafts() {
  const reads: { id: string; answer: (view: View | null | undefined) => void }[] = [];
  const service = new ViewDraftService(
    (id) => new Promise<View | null | undefined>((resolve) => reads.push({ id, answer: resolve }))
  );
  /** The view's draft as get gives it once what is out settled, without waiting for a read nobody answers. */
  const got = async () => {
    const draft = track(service.get(V));
    await settled();
    return draft;
  };
  /** Seeds the view's draft from a read answered with view. */
  const seed = async (view: View = saved) => {
    const draft = track(service.get(V));
    reads.at(-1)!.answer(view);
    await settled();
    return draft;
  };
  return { reads, service, got, seed };
}

describe("a view's draft", () => {
  it("is seeded by one read of the view, and read no more while it lasts", async () => {
    const { reads, got, seed } = drafts();
    expect((await seed()).value).toEqual(fieldsOf(saved));
    expect(reads.map(({ id }) => id)).toEqual([V]);
    const again = await got();
    expect(reads, "the draft held, nothing is read").toHaveLength(1);
    expect(again.value).toEqual(fieldsOf(saved));
  });

  it("changes only the fields a change names, the rest as seeded, and sends nothing", async () => {
    const { reads, service, got, seed } = drafts();
    await seed();
    const high = { priority: ["high"] };
    await expect(service.update(V, { filters: high })).resolves.toEqual({ ...fieldsOf(saved), filters: high });
    const hidden = { ...saved.display_properties, state: false };
    const whole = { filters: high, display_filters: saved.display_filters, display_properties: hidden };
    await expect(service.update(V, { display_properties: hidden })).resolves.toEqual(whole);
    expect((await got()).value).toEqual(whole);
    expect(reads).toHaveLength(1);
  });

  it("refuses a change of a view whose draft is not held, and makes none of it", async () => {
    const { reads, service, seed } = drafts();
    await expect(service.update(V, { filters: { priority: ["high"] } })).rejects.toThrow(V);
    expect(reads).toHaveLength(0);
    expect((await seed()).value, "no draft was made: the view is read").toEqual(fieldsOf(saved));
    expect(reads).toHaveLength(1);
  });

  it("takes nerve's answer to Update view while it lasts, and reads nothing for it", async () => {
    const { reads, service, got, seed } = drafts();
    await seed();
    service.adopt(V, changed);
    const draft = await got();
    expect(reads).toHaveLength(1);
    expect(draft.value).toEqual(fieldsOf(changed));
  });

  it("takes no answer once the page ended it: the next page reads the view again, once", async () => {
    const { reads, service, seed } = drafts();
    await seed();
    service.discard(V);
    service.adopt(V, changed);
    const draft = await seed();
    expect(reads, "the answer made no draft: the view is read again").toHaveLength(2);
    expect(draft.value).toEqual(fieldsOf(saved));
  });

  it("ends with a seed then out, whose answer makes no draft: the next page reads the view again, once", async () => {
    const { reads, service, seed } = drafts();
    const late = track(service.get(V));
    service.discard(V);
    reads[0]!.answer(saved);
    await settled();
    expect(late.error, "the seed answered after the page left").toBeInstanceOf(Error);
    expect(late.value).toBeUndefined();
    const next = await seed(changed);
    expect(reads, "no draft was made: the view is read again").toHaveLength(2);
    expect(next.value).toEqual(fieldsOf(changed));
  });

  it.each([
    ["null", null],
    ["undefined", undefined],
  ])("makes none of a read that gives no view (%s): the next page reads it again", async (_, answer) => {
    const { reads, service, got } = drafts();
    const none = track(service.get(V));
    reads[0]!.answer(answer);
    await settled();
    expect(none.error).toBeInstanceOf(Error);
    await got();
    expect(reads).toHaveLength(2);
  });
});
