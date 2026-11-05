/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { AttachmentItem, type AttachmentRow } from "./item";
export function AttachmentList({
  rows,
  download,
  remove,
}: {
  rows: readonly AttachmentRow[];
  download?(id: string): Promise<void>;
  remove?(row: AttachmentRow): void;
}) {
  return (
    <ul className="space-y-2">
      {rows.map((row) => (
        <AttachmentItem key={row.id} row={row} download={download} remove={remove} />
      ))}
    </ul>
  );
}
