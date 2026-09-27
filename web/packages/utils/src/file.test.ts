// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, describe, expect, it, vi } from "vitest";
import { csvDownload, csvText } from "./file";

// The CSV a new personal access token or webhook secret downloads with (M2 design 7.7): its fields are names,
// descriptions and dates people write, so the text follows RFC 4180, and the file holds all of it.

describe("csvText", () => {
  it("writes a plain record as its fields apart by commas, ending with CRLF", () => {
    expect(csvText([["a", "b", "c"]])).toBe("a,b,c\r\n");
    expect(
      csvText([
        ["a", "b"],
        ["c", "d"],
      ])
    ).toBe("a,b\r\nc,d\r\n");
  });

  it("quotes a field with a comma", () => {
    expect(csvText([["ci, deploy", "b"]])).toBe('"ci, deploy",b\r\n');
  });

  it("quotes a field with a double quote, and doubles its quotes", () => {
    expect(csvText([['the "ci" token', "b"]])).toBe('"the ""ci"" token",b\r\n');
    expect(csvText([['"', "b"]])).toBe('"""",b\r\n');
  });

  it("quotes a field with a line break, CR or LF", () => {
    expect(csvText([["line 1\r\nline 2", "b"]])).toBe('"line 1\r\nline 2",b\r\n');
    expect(csvText([["line 1\nline 2", "b"]])).toBe('"line 1\nline 2",b\r\n');
    expect(csvText([["line 1\rline 2", "b"]])).toBe('"line 1\rline 2",b\r\n');
  });

  it("keeps a # as it is", () => {
    expect(csvText([["ci #1", "#"]])).toBe("ci #1,#\r\n");
  });

  it("keeps an empty field", () => {
    expect(csvText([["a", "", "c"]])).toBe("a,,c\r\n");
    expect(csvText([["", ""]])).toBe(",\r\n");
  });

  it("writes an object as a header record of its keys and a record of its values", () => {
    expect(csvText({ Title: "ci, #1", Description: "", Expiry: "Never expires", "Secret key": "nrv_secret" })).toBe(
      'Title,Description,Expiry,Secret key\r\n"ci, #1",,Never expires,nrv_secret\r\n'
    );
  });
});

describe("csvDownload", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("downloads the whole text in a Blob's URL, which it lets go of after the click", async () => {
    // jsdom has no object URLs and does not follow links: these record what the page asks of the browser
    const events: string[] = [];
    const blobs = new Map<string, Blob>();
    vi.spyOn(URL, "createObjectURL").mockImplementation((blob) => {
      const url = `blob:nerve/${blobs.size}`;
      blobs.set(url, blob as Blob);
      events.push(`create ${url}`);
      return url;
    });
    vi.spyOn(URL, "revokeObjectURL").mockImplementation((url) => events.push(`revoke ${url}`));
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) {
      events.push(`click ${this.getAttribute("href")} ${this.download} ${this.isConnected}`);
    });

    csvDownload({ Title: "ci, #1", "Secret key": "nrv_secret" }, "secret-key-1");

    expect(events).toEqual(["create blob:nerve/0", "click blob:nerve/0 secret-key-1.csv true", "revoke blob:nerve/0"]);
    const blob = blobs.get("blob:nerve/0");
    expect(blob?.type).toBe("text/csv;charset=utf-8");
    expect(await blob?.text()).toBe('Title,Secret key\r\n"ci, #1",nrv_secret\r\n');
    expect(document.querySelector("a")).toBeNull();
  });
});
