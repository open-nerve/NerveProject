import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { activityPage, descriptionPage, descriptionVersion, updateIssue } from "../../fixtures/issues";
import { filePixel, readFile, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F12 (API): replacing, reordering and repeating image identities cannot skip history; block IDs retain migration semantics and removed images remain readable",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
    await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
    const image = () =>
      storedFile(
        api,
        w.owner.token,
        "ISSUE_DESCRIPTION",
        { kind: "issue", id: w.issue.id },
        filePixel,
        "image/png",
        "same.png",
        uploadStatus
      );
    const a = await image(),
      b = await image();
    const initial = await updateIssue(api, w.owner.token, w.issue.id, { description_html: imageHTML(a.id, b.id) });
    const bound = await assetRows(db, [a.id, b.id]);
    const versions = () => descriptionPage(api, w.owner.token, w.issue.id, undefined, 100);
    const activities = async () =>
      (await activityPage(api, w.owner.token, w.issue.id, undefined, 100)).data.filter(
        ({ field }) => field === "description"
      );
    const beforeVersions = await versions(),
      beforeActivities = await activities();
    const migrated = await updateIssue(api, w.owner.token, w.issue.id, {
      description_html: imageHTML(a.id, b.id).replace("<p>", '<p data-id="body">'),
      skip_activity: true,
    });
    expect(migrated.updated_at).toBe(initial.updated_at);
    expect(await versions()).toEqual(beforeVersions);
    expect(await activities()).toEqual(beforeActivities);
    let versionCount = beforeVersions.data.length,
      activityCount = beforeActivities.length;
    const changed = [imageHTML(b.id, a.id), imageHTML(b.id, b.id, a.id), imageHTML(a.id), "<p>Body</p>"];
    for (const [index, html] of changed.entries()) {
      // Alternate real authors so M4's legitimate version/activity merge window
      // cannot hide whether this image-only save produced both persisted facts.
      const actor = index % 2 === 0 ? w.other : w.owner;
      // eslint-disable-next-line no-await-in-loop -- Every transition changes only image identity, not plain text.
      const saved = await updateIssue(api, actor.token, w.issue.id, { description_html: html, skip_activity: true });
      expect(saved.description_html).toBe(html);
      // eslint-disable-next-line no-await-in-loop
      const page = await versions();
      expect(page.data).toHaveLength(++versionCount);
      // eslint-disable-next-line no-await-in-loop
      expect(await activities()).toHaveLength(++activityCount);
      // eslint-disable-next-line no-await-in-loop
      expect(await assetRows(db, [a.id, b.id])).toEqual(bound);
    }
    const histories = await versions();
    let sawBoth = false;
    for (const row of histories.data) {
      // eslint-disable-next-line no-await-in-loop
      const saved = await descriptionVersion(api, w.other.token, row.id);
      if (saved.description_html.includes(a.id) && saved.description_html.includes(b.id)) sawBoth = true;
    }
    expect(sawBoth).toBe(true);
    expect(await readFile(api, w.other.token, a.id)).toEqual(filePixel);
    expect(await readFile(api, w.other.token, b.id)).toEqual(filePixel);
  }
);
