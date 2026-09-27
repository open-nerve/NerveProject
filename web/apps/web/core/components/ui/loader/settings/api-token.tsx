/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { range } from "lodash-es";

/** The rows of a list of tokens, while it loads. */
export function APITokenSettingsLoader() {
  return (
    <div className="divide-y-[0.5px] divide-subtle-1">
      {range(2).map((i) => (
        <div key={i} className="flex flex-col gap-2 py-3">
          <div className="flex items-center gap-2">
            <span className="h-5 w-28 rounded-sm bg-layer-1" />
            <span className="h-5 w-16 rounded-sm bg-layer-1" />
          </div>
          <span className="h-5 w-36 rounded-sm bg-layer-1" />
        </div>
      ))}
    </div>
  );
}
