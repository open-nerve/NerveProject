/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/core";
import { afterEach, expect, it, vi } from "vitest";
import { convertHTMLToMarkdown } from "@nerve/utils";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import { useEditor } from "@/hooks/use-editor";
import { createEditor, destroyEditors, getEditorMetaData } from "./editor-test-fixture";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;

afterEach(() => {
  destroyEditors();
  document.body.replaceChildren();
});

it.each([
  {
    name: "literal Markdown punctuation",
    html: "<p>*literal* [label](https://example.com) &lt;not html&gt;</p>",
  },
  {
    name: "nested inline marks",
    html: "<p><em>Italic <code>keep()</code></em> <u><strong>underlined bold</strong></u></p>",
  },
])("copies $name without losing or reinterpreting text", async ({ html }) => {
  const original = await createEditor({ content: html });
  const markdown = getEditorRefHelpers({ editor: original, getEditorMetaData }).getMarkDown();
  const pasted = await createEditor();
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { getData: (type: string) => (type === "text/plain" ? markdown : "") },
  });
  pasted.view.dom.dispatchEvent(event);
  // Markdown ends in a block newline; the inline paste parser preserves it as trailing whitespace.
  expect(pasted.getText().trimEnd()).toBe(original.getText());
  if (html.includes("<code>")) {
    // The editor's code mark excludes other marks, so it is not nested under emphasis after parsing.
    expect(pasted.view.dom.querySelector("code")?.textContent).toBe("keep()");
    expect(pasted.view.dom.querySelector("strong")?.textContent).toBe("underlined bold");
  } else {
    expect(pasted.view.dom.querySelector("em")).toBeNull();
  }
});

it("preserves nested code and bold text when exporting DOM marks", () => {
  const markdown = convertHTMLToMarkdown({
    description_html: "<p><em>Italic <code>keep()</code></em> <u><strong>underlined bold</strong></u></p>",
    metaData: getEditorMetaData(),
  });
  expect(markdown).toContain("`keep()`");
  expect(markdown).toContain("**underlined bold**");
});

it("exports custom references as Markdown nodes without interpreting their labels", async () => {
  const markdown = convertHTMLToMarkdown({
    description_html:
      '<p><mention-component entity_name="user_mention" entity_identifier="user-ada"></mention-component></p>',
    metaData: {
      file_assets: [],
      user_mentions: [{ id: "user-ada", display_name: "*Ada* [team]", url: "https://example.com/ada" }],
    },
  });
  const pasted = await createEditor();
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { getData: (type: string) => (type === "text/plain" ? markdown : "") },
  });
  pasted.view.dom.dispatchEvent(event);
  expect(pasted.view.dom.querySelector("a")?.textContent).toBe("@*Ada* [team]");
  expect(pasted.view.dom.querySelector("em")).toBeNull();
});

it.each([false, true])("copies a task with checked=%s without adding an empty paragraph", async (checked) => {
  const original = await createEditor({
    content: `<ul data-type="taskList"><li data-type="taskItem" data-checked="${checked}"><p>Keep task</p></li></ul>`,
  });
  expect(original.view.dom.querySelectorAll("li p")).toHaveLength(1);
  const markdown = getEditorRefHelpers({ editor: original, getEditorMetaData }).getMarkDown();
  const pasted = await createEditor();
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { getData: (type: string) => (type === "text/plain" ? markdown : "") },
  });
  pasted.view.dom.dispatchEvent(event);
  // The task parser retains the separator after its checkbox; it must not create a paragraph or visible marker.
  expect(pasted.getText().trim()).toBe("Keep task");
  expect(pasted.view.dom.querySelectorAll("li p")).toHaveLength(1);
  expect(pasted.view.dom.querySelector("li")?.getAttribute("data-checked")).toBe(String(checked));
});

it("the consumer hook parses pasted Markdown once and still accepts intentional formatting", async () => {
  const current: { editor: Editor | null } = { editor: null };
  const mentionHandler = { getMentionedEntityDetails: () => undefined, renderComponent: () => null };
  function Host() {
    const editor = useEditor({
      id: "markdown-paste",
      editable: true,
      disabledExtensions: ["image"],
      enableHistory: true,
      initialValue: "",
      getEditorMetaData,
      mentionHandler,
    });
    current.editor = editor;
    return createElement(EditorContent, { editor });
  }
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () => root.render(createElement(Host)));
    await vi.waitFor(() => expect(current.editor).not.toBeNull());
    const editor = current.editor!;
    const paste = (text: string) => {
      const event = new Event("paste", { bubbles: true, cancelable: true });
      Object.defineProperty(event, "clipboardData", {
        value: { getData: (type: string) => (type === "text/plain" ? text : "") },
      });
      editor.view.dom.dispatchEvent(event);
    };
    await act(async () => paste("\\*literal\\* \\*\\*bold\\*\\* \\~\\~strike\\~\\~ \\`code\\`"));
    expect(editor.getText()).toBe("*literal* **bold** ~~strike~~ `code`");
    expect(editor.view.dom.querySelector("em,strong,s,code")).toBeNull();
    await act(async () => editor.commands.clearContent());
    await act(async () => paste("**Bold** and _italic_ and ~~strike~~ and `code`"));
    expect(editor.view.dom.querySelector("strong")?.textContent).toBe("Bold");
    expect(editor.view.dom.querySelector("em")?.textContent).toBe("italic");
    expect(editor.view.dom.querySelector("s")?.textContent).toBe("strike");
    expect(editor.view.dom.querySelector("code")?.textContent).toBe("code");
    await act(async () => editor.commands.clearContent());
    await act(async () => paste("Visit https://example.com and :smile:"));
    expect(editor.view.dom.querySelector("a")?.getAttribute("href")).toBe("https://example.com");
    expect(editor.view.dom.querySelector('[data-type="emoji"]')).not.toBeNull();
  } finally {
    await act(async () => root.unmount());
  }
});

it("keeps separate paragraphs and nested tasks while exporting a task list", async () => {
  const original = await createEditor({
    content:
      '<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Parent</p><p>Second paragraph</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Child</p></li></ul></li></ul>',
  });
  const markdown = getEditorRefHelpers({ editor: original, getEditorMetaData }).getMarkDown();
  const pasted = await createEditor();
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { getData: (type: string) => (type === "text/plain" ? markdown : "") },
  });
  pasted.view.dom.dispatchEvent(event);
  expect(Array.from(pasted.view.dom.querySelectorAll("p"), (node) => node.textContent?.trim())).toEqual([
    "Parent",
    "Second paragraph",
    "Child",
  ]);
  expect(Array.from(pasted.view.dom.querySelectorAll("li"), (node) => node.getAttribute("data-checked"))).toEqual([
    "true",
    "false",
  ]);
  expect(pasted.view.dom.querySelectorAll("li li")).toHaveLength(1);
});
