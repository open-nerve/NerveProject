import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectModuleLinks, moduleLinkMoments, type ModuleLinkRow } from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import {
  createModule,
  createModuleLink,
  expectModuleLinkCrawled,
  listModuleLinks,
  moduleCalls,
  moduleLinkCrawls,
  updateModuleLink,
} from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G10, a module's links (M6 design 3.2, M6 P2 spec 2.3, R11): an address checked and given its scheme as a work
// item's link's; once among a module's links; its metadata crawled in the background, by the work items' crawl, and
// emptied when the address changes. Every address is link-local, so each crawl is blocked and nothing leaves the
// machine. The admin calls with a personal access token, the member with the token of his sign-in.

/** The metadata of a crawl of url that was blocked, as a link-local address is. */
function blocked(url: string) {
  return { error: "blocked", url };
}

test("G10 (API): Web's member adds Docs a link by a bare address and an empty title, the admin one by its https address, each with empty metadata until the background crawl, blocked; another scheme, credentials, a blank address, a title with NUL, Docs's address again and a guest's link are refused, while Guide takes the same address; the links are listed, the last first, alike to either token and to the guest; a title alone is changed, the metadata kept and nothing crawled; another link's address and a guest's change are refused; a new address empties the metadata until it is crawled; a deleted link is not found; Docs archived, and Web's modules off, refuse each write of a link while the links are read", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const [mem, guest] = await Promise.all(["mem", "guest"].map((label) => newAccount(api, testInfo, label)));
  if (!mem || !guest) {
    throw new Error("the accounts were not registered");
  }
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  const turn = async (on: boolean) => {
    const changed = await changeProject(api, admin.token, web.id, { module_view: on });
    expect(changed.status, `the admin turns Web's modules ${on ? "on" : "off"}: ${JSON.stringify(changed)}`).toBe(200);
  };
  await turn(true);
  const browser = (await login(api, mem.email)).access_token;
  const { addLink, changeLink, removeLink, update, archive } = moduleCalls(api);
  const docs = await createModule(api, admin.token, web.id, { name: "Docs" });
  const guide = await createModule(api, browser, web.id, { name: "Guide" });

  // mem adds Docs a link by a bare address within spaces and an empty title: the address given its scheme, the empty
  // title kept, the metadata empty; its crawl, enqueued with it, then writes what kept it from the page.
  const specURL = "http://169.254.169.254/g10/spec";
  const spec = await createModuleLink(api, browser, docs.id, { url: "  169.254.169.254/g10/spec  ", title: "" });
  expect(
    [spec.module_id, spec.url, spec.title, spec.metadata, spec.created_by_id],
    "the spec link as mem adds it"
  ).toEqual([docs.id, specURL, "", {}, mem.id]);
  await expectModuleLinkCrawled(db, spec.id, [specURL]);
  // The admin adds Docs a link by its https address, without a title.
  const designURL = "https://169.254.169.254/g10/design";
  const design = await createModuleLink(api, admin.token, docs.id, { url: designURL });
  expect([design.url, design.title, design.metadata, design.created_by_id], "the design link as added").toEqual([
    designURL,
    null,
    {},
    admin.id,
  ]);
  await expectModuleLinkCrawled(db, design.id, [designURL]);
  let docsLinks: ModuleLinkRow[] = [
    { url: specURL, title: "", adder: mem.email, by: mem.email, removed: null },
    { url: designURL, title: null, adder: admin.email, by: admin.email, removed: null },
  ];
  await expectModuleLinks(db, docs.id, docsLinks);

  // Refused, each changing nothing, as a work item's link: an address of another scheme; one with credentials; a
  // blank one; a title with NUL; the spec link's address, bare, again; the guest's link.
  expect(
    [
      await addLink(browser, docs.id, { url: "ftp://169.254.169.254/g10/file" }),
      await addLink(browser, docs.id, { url: "http://user:secret@169.254.169.254/g10/private" }),
      await addLink(browser, docs.id, { url: "   " }),
      await addLink(browser, docs.id, { url: "169.254.169.254/g10/notes", title: "Notes\u0000" }),
      await addLink(admin.token, docs.id, { url: "169.254.169.254/g10/spec" }),
      await addLink(guest.token, docs.id, { url: "169.254.169.254/g10/guest" }),
    ],
    "an ftp address, credentials, a blank address, a NUL title, the spec address again, the guest's link"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "url", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "url", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "url", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "title", code: "invalid_format" }] },
    { status: 409, code: "planning.module_link_exists" },
    { status: 403, code: "forbidden" },
  ]);
  await expectModuleLinks(db, docs.id, docsLinks);
  // The same address is another module's to have: mem adds it to Guide.
  const guideSpec = await createModuleLink(api, browser, guide.id, { url: specURL, title: "Spec" });
  await expectModuleLinkCrawled(db, guideSpec.id, [specURL]);
  const guideLinks: ModuleLinkRow[] = [{ url: specURL, title: "Spec", adder: mem.email, by: mem.email, removed: null }];
  await expectModuleLinks(db, guide.id, guideLinks);

  // Docs's links, the last added first, with their crawls' metadata: the same to mem's two tokens and to the guest,
  // who reads them.
  const listed = await listModuleLinks(api, browser, docs.id);
  expect(
    listed.map((l) => [l.url, l.title, l.metadata]),
    "Docs's links as mem lists them"
  ).toEqual([
    [designURL, null, blocked(designURL)],
    [specURL, "", blocked(specURL)],
  ]);
  expect(await listModuleLinks(api, mem.token, docs.id), "Docs's links to mem's personal access token").toEqual(listed);
  expect(await listModuleLinks(api, guest.token, docs.id), "Docs's links to the guest").toEqual(listed);

  // mem titles the spec link, its address not given: the metadata is kept and nothing is crawled; the admin clears
  // the title.
  const titled = await updateModuleLink(api, browser, spec.id, { title: "Spec" });
  expect([titled.url, titled.title, titled.metadata], "the spec link titled").toEqual([
    specURL,
    "Spec",
    blocked(specURL),
  ]);
  expect(await changeLink(admin.token, spec.id, { title: null }), "the admin clears its title").toEqual({
    status: 200,
  });
  expect(await moduleLinkCrawls(db, spec.id), "the spec link's crawls after its titles").toEqual([
    { url: specURL, state: "completed" },
  ]);
  docsLinks = [{ ...docsLinks[0]!, title: null, by: admin.email }, docsLinks[1]!];
  await expectModuleLinks(db, docs.id, docsLinks);

  // Refused, each changing nothing: the design link given the spec link's address; an address of another scheme;
  // the guest's change.
  expect(
    [
      await changeLink(admin.token, design.id, { url: specURL }),
      await changeLink(browser, design.id, { url: "file:///etc/passwd" }),
      await changeLink(guest.token, design.id, { title: "Guest's" }),
    ],
    "the design link given the spec address, a file address, the guest's change"
  ).toEqual([
    { status: 409, code: "planning.module_link_exists" },
    { status: 422, code: "validation_failed", errors: [{ field: "url", code: "not_allowed" }] },
    { status: 403, code: "forbidden" },
  ]);
  await expectModuleLinks(db, docs.id, docsLinks);

  // mem moves the spec link to its second version by a bare address: the metadata is emptied at once, and the crawl of
  // the new address, enqueued with the change, writes its own.
  const specV2URL = "http://169.254.169.254/g10/spec-v2";
  const moved = await updateModuleLink(api, browser, spec.id, { url: "169.254.169.254/g10/spec-v2" });
  expect([moved.url, moved.title, moved.metadata], "the spec link moved").toEqual([specV2URL, null, {}]);
  await expectModuleLinkCrawled(db, spec.id, [specURL, specV2URL]);
  expect(
    (await listModuleLinks(api, browser, docs.id)).map((l) => [l.url, l.metadata]),
    "Docs's links after the move"
  ).toEqual([
    [designURL, blocked(designURL)],
    [specV2URL, blocked(specV2URL)],
  ]);
  docsLinks = [{ ...docsLinks[0]!, url: specV2URL, by: mem.email }, docsLinks[1]!];
  await expectModuleLinks(db, docs.id, docsLinks);

  // The admin deletes the design link, at its last write, by him: it is listed no more, and changed or deleted again
  // it is not found.
  expect(await removeLink(admin.token, design.id), "the admin deletes the design link").toEqual({ status: 204 });
  docsLinks = [docsLinks[0]!, { ...docsLinks[1]!, removed: (await moduleLinkMoments(db, docs.id))[designURL] ?? "" }];
  await expectModuleLinks(db, docs.id, docsLinks);
  expect(
    [await changeLink(browser, design.id, { title: "Design" }), await removeLink(admin.token, design.id)],
    "the deleted design link changed, and deleted again"
  ).toEqual(Array(2).fill({ status: 404, code: "planning.module_link_not_found" }));
  expect(
    (await listModuleLinks(api, browser, docs.id)).map((l) => l.url),
    "Docs's links after the deletion"
  ).toEqual([specV2URL]);

  // The admin completes Docs and archives it: its links are read as before, and none is added, changed, even by
  // nothing, or deleted (M6 P2 spec R2).
  expect(await update(admin.token, docs.id, { status: "completed" }), "the admin completes Docs").toEqual({
    status: 200,
  });
  expect(await archive(admin.token, docs.id), "the admin archives Docs").toEqual({ status: 200 });
  const archived = await listModuleLinks(api, browser, docs.id);
  expect(
    [
      await addLink(browser, docs.id, { url: "169.254.169.254/g10/later" }),
      await changeLink(browser, spec.id, { title: "Spec 2" }),
      await changeLink(admin.token, spec.id, {}),
      await removeLink(admin.token, spec.id),
    ],
    "archived Docs's link added, changed, changed by nothing and deleted"
  ).toEqual(Array(4).fill({ status: 409, code: "planning.archived" }));
  await expectModuleLinks(db, docs.id, docsLinks);
  expect(await listModuleLinks(api, mem.token, docs.id), "archived Docs's links").toEqual(archived);

  // Web's modules off: no link is added, changed or deleted, Guide's or archived Docs's, the switch answering first;
  // Guide's links are read as before.
  const guideBefore = await listModuleLinks(api, browser, guide.id);
  await turn(false);
  expect(
    [
      await addLink(browser, guide.id, { url: "169.254.169.254/g10/off" }),
      await changeLink(browser, guideSpec.id, { title: "Off" }),
      await removeLink(admin.token, guideSpec.id),
      await addLink(admin.token, docs.id, { url: "169.254.169.254/g10/off" }),
    ],
    "Guide's link added, changed and deleted, and archived Docs's added, modules off"
  ).toEqual(Array(4).fill({ status: 409, code: "planning.modules_disabled" }));
  await expectModuleLinks(db, guide.id, guideLinks);
  await expectModuleLinks(db, docs.id, docsLinks);
  expect(await listModuleLinks(api, browser, guide.id), "Guide's links, modules off").toEqual(guideBefore);
});
