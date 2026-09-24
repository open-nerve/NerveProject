/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// local imports
import { NerveLogo } from "./nerve-logo";

// The mark reads on light and dark backgrounds alike, so the markup does not depend on the theme and hydrates
// cleanly where it is prerendered (HydrateFallback in app/root.tsx).
export function LogoSpinner() {
  return (
    <div className="flex items-center justify-center">
      <NerveLogo className="h-6 w-auto animate-pulse sm:h-11" />
    </div>
  );
}
