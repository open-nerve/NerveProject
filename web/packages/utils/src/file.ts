/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/** A CSV field (RFC 4180): quoted, its quotes doubled, when it holds a comma, a double quote, a CR or an LF. */
const csvField = (field: string): string => (/[",\r\n]/.test(field) ? `"${field.replaceAll('"', '""')}"` : field);

/**
 * @description the text of a CSV file (RFC 4180): fields apart by commas, each record ending with CRLF
 * @param {Array<Array<string>> | { [key: string]: string }} data - the records, or one record with its field names
 * as the header record
 * @returns {string} the CSV text
 */
export const csvText = (data: Array<Array<string>> | { [key: string]: string }): string => {
  const rows = Array.isArray(data) ? data : [Object.keys(data), Object.values(data)];
  return rows.map((row) => `${row.map(csvField).join(",")}\r\n`).join("");
};

/**
 * @description downloads a CSV file with the whole text: as a Blob, not in a data: URL, where a "#" would end
 * the file
 * @param {Array<Array<string>> | { [key: string]: string }} data - The data to be exported to CSV
 * @param {string} name - The name of the file to be downloaded
 */
export const csvDownload = (data: Array<Array<string>> | { [key: string]: string }, name: string) => {
  const url = URL.createObjectURL(new Blob([csvText(data)], { type: "text/csv;charset=utf-8" }));

  const link = document.createElement("a");
  link.href = url;
  link.download = `${name}.csv`;

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  // The click has resolved the URL to its Blob (a blob: URL is resolved as the link's URL is parsed), so the
  // download no longer needs the URL.
  URL.revokeObjectURL(url);
};
