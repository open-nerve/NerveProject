import { readFileSync } from "node:fs";
import { Editor, type JSONContent } from "@tiptap/core";
import { afterEach, expect, it } from "vitest";
import { CoreEditorExtensions } from "@/extensions/extensions";
import { CoreEditorProps } from "@/props";
import type { TFileHandler, TMentionHandler } from "@/types";

const fileHandler: TFileHandler = {
  assetsUploadStatus: {},
  cancel: () => {},
  checkIfAssetExists: async () => true,
  delete: async () => {},
  getAssetDownloadSrc: async (path) => path,
  getAssetSrc: async (path) => path,
  restore: async () => {},
  upload: async () => "asset-id",
  duplicate: async () => "asset-id",
  validation: { maxFileSize: 5 * 1024 * 1024 },
};
const mentionHandler: TMentionHandler = {
  getMentionedEntityDetails: () => ({ display_name: "Ada" }),
  renderComponent: () => null,
};
const editors: Editor[] = [];
afterEach(() => {
  for (const editor of editors.splice(0)) editor.destroy();
  document.body.replaceChildren();
});
const createEditor = async (content: string | JSONContent, editable = false) => {
  const element = document.createElement("div");
  document.body.append(element);
  const editor = new Editor({
    element,
    editable,
    parseOptions: { preserveWhitespace: true },
    editorProps: CoreEditorProps({ editorClassName: "" }),
    extensions: CoreEditorExtensions({
      disabledExtensions: [],
      editable,
      enableHistory: true,
      fileHandler,
      mentionHandler,
      getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
    }).map((extension) =>
      extension.name === "uniqueID"
        ? extension.configure({
            generateUniqueID: ({ node, pos }: { node: { type: { name: string } }; pos: number }) =>
              `fixed-${node.type.name}-${pos}`,
          })
        : extension
    ),
    content,
  });
  editors.push(editor);
  await new Promise((resolve) => editor.on("create", resolve));
  return editor;
};

type Fixture = {
  name: string;
  input: string;
  sanitized: string;
  reserialized: string;
  editor_document?: JSONContent;
  preserved_elements?: {
    selector: string;
    attributes?: Record<string, string>;
    styles?: Record<string, string>;
    text?: string;
  }[];
  editable?: boolean;
};
const fixtures = JSON.parse(readFileSync("../../../api/testdata/issue-richtext.json", "utf8")) as { cases: Fixture[] };

it("keeps every saved node and mark through the real editor schema", async () => {
  const nodes = new Set<string>();
  const marks = new Set<string>();
  for (const fixture of fixtures.cases) {
    // oxlint-disable-next-line no-await-in-loop -- real editors share document plugins; finish one fixture at a time
    const editor = await createEditor(fixture.sanitized, fixture.editable);
    if (fixture.editor_document) {
      // oxlint-disable-next-line no-await-in-loop -- compare this source document with the current fixture editor
      const source = await createEditor(fixture.editor_document, fixture.editable);
      expect(source.getHTML(), "fixture must originate in the real editor").toBe(fixture.input);
      source.state.doc.descendants((node) => {
        for (const mark of node.marks) marks.add(mark.type.name);
      });
      expect(source.schema.marks.textStyle.spec.attrs ?? {}).toEqual({});
      source.view.dispatch(source.state.tr.removeMark(0, source.state.doc.content.size, source.schema.marks.textStyle));
      expect(editor.getJSON(), "only attribute-free textStyle normalizes away").toEqual(source.getJSON());
    }
    const body = new DOMParser().parseFromString(editor.getHTML(), "text/html").body;
    for (const kept of fixture.preserved_elements ?? []) {
      const element = body.querySelector<HTMLElement>(kept.selector);
      expect(element, kept.selector).not.toBeNull();
      for (const [key, value] of Object.entries(kept.attributes ?? {}))
        expect(element?.getAttribute(key), kept.selector + "/" + key).toBe(value);
      for (const [key, value] of Object.entries(kept.styles ?? {}))
        expect(element?.style[key as keyof CSSStyleDeclaration], kept.selector + "/" + key).toBe(value);
      if (kept.text !== undefined) expect(element?.textContent, kept.selector).toBe(kept.text);
    }
    expect(editor.getHTML(), fixture.name).toBe(fixture.reserialized);
    editor.state.doc.descendants((node) => {
      nodes.add(node.type.name);
      for (const mark of node.marks) marks.add(mark.type.name);
    });
    nodes.add("doc");
  }
  const schema = editors[0].schema;
  expect([...nodes].toSorted()).toEqual(Object.keys(schema.nodes).toSorted());
  expect([...marks].toSorted()).toEqual(Object.keys(schema.marks).toSorted());
});

it("reopens emoji nodes instead of swallowing them as color spans", async () => {
  const editor = await createEditor('<p><span data-name="smile" data-type="emoji">😄</span></p>');
  const emoji = editor.state.doc.firstChild?.firstChild;
  expect(emoji?.type.name).toBe("emoji");
  expect(emoji?.attrs.name).toBe("smile");
  expect(editor.getHTML()).toBe(
    '<p class="editor-paragraph-block"><span data-name="smile" data-type="emoji">😄</span></p>'
  );
});

it("reopens both custom colors when the saved span also has CSS", async () => {
  const editor = await createEditor(
    '<p><span data-text-color="#123456" data-background-color="#abcdef" style="color: rgb(18, 52, 86); background-color: rgb(171, 205, 239)">Color</span></p>'
  );
  const marks = editor.state.doc.firstChild?.firstChild?.marks;
  expect(marks?.map((mark) => ({ type: mark.type.name, attrs: { ...mark.attrs } }))).toEqual([
    { type: "customColor", attrs: { color: "#123456", backgroundColor: "#abcdef" } },
  ]);
  const element = new DOMParser().parseFromString(editor.getHTML(), "text/html").querySelector("span");
  expect(element?.getAttribute("data-text-color")).toBe("#123456");
  expect(element?.getAttribute("data-background-color")).toBe("#abcdef");
  expect(element?.style.color).toBe("rgb(18, 52, 86)");
  expect(element?.style.backgroundColor).toBe("rgb(171, 205, 239)");
});

it("normalizes the current attribute-free textStyle without losing text", async () => {
  const editor = await createEditor('<p><span style="color: red">Style</span></p>');
  expect(editor.schema.marks.textStyle.spec.attrs ?? {}).toEqual({});
  expect(editor.state.doc.firstChild?.firstChild?.marks.map((mark) => mark.type.name)).toEqual(["textStyle"]);
  expect(editor.getHTML()).toBe('<p class="editor-paragraph-block"><span>Style</span></p>');
  const reopened = await createEditor(editor.getHTML());
  expect(reopened.getHTML()).toBe('<p class="editor-paragraph-block">Style</p>');
  expect(reopened.getText()).toBe("Style");
  expect(reopened.state.doc.firstChild?.firstChild?.marks).toEqual([]);
});

it("cannot regenerate injected CSS from sanitized color attributes", async () => {
  const fixture = fixtures.cases.find((item) => item.name === "color-attribute-injection");
  expect(fixture).toBeDefined();
  const editor = await createEditor(fixture!.sanitized);
  const body = new DOMParser().parseFromString(editor.getHTML(), "text/html").body;
  expect(body.querySelector("[data-text-color], [data-background-color], [textcolor]")).toBeNull();
  for (const element of body.querySelectorAll<HTMLElement>("[style]")) expect(element.getAttribute("style")).toBe("");
  expect(editor.getHTML()).not.toMatch(/position|url\(/);
});
