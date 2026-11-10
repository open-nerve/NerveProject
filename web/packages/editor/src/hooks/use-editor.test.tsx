/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createElement, useLayoutEffect } from "react";
import { createRoot } from "react-dom/client";
import { expect, it } from "vitest";
import type { EditorRefApi } from "@/types";
import { getEditorMetaData } from "../editor-test-fixture";
import { useEditor } from "./use-editor";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;

// TipTap builds the editor in an effect after the first commit (immediatelyRender: false), and a native listener can
// run in between: Headless UI's Escape on the window, before React's passive effects. A host that reads the ref then
// must find no editor, not an API over a missing one whose isEditorReadyToDiscard says false as for an unfinished image.
it("exposes the ref only once its editor exists", async () => {
  const ref: { current: EditorRefApi | null } = { current: null };
  const commits: (EditorRefApi | null)[] = [];
  function Host() {
    useEditor({
      disabledExtensions: [],
      editable: true,
      enableHistory: true,
      forwardedRef: ref,
      id: "use-editor-test",
      getEditorMetaData,
      initialValue: "<p></p>",
      mentionHandler: { getMentionedEntityDetails: () => undefined, renderComponent: () => null },
    });
    // after useEditor's own layout work in each commit: what a native listener reads before the next effects run
    useLayoutEffect(() => {
      commits.push(ref.current);
    });
    return null;
  }
  const root = createRoot(document.body.appendChild(document.createElement("div")));
  await act(async () => root.render(createElement(Host)));

  expect(commits[0]).toBeNull();
  expect(commits.at(-1)).not.toBeNull();
  expect(ref.current?.isEditorReadyToDiscard()).toBe(true);

  await act(async () => root.unmount());
  expect(ref.current).toBeNull();
});
