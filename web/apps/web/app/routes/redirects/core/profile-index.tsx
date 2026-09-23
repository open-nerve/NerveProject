/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { redirect } from "react-router";
import type { Route } from "./+types/profile-index";

// The profile page itself has no content of its own any more (M1 design 3.2): it redirects to the
// "assigned" tab, so every existing link to /:workspaceSlug/profile/:userId keeps working.
export const clientLoader = ({ params, request }: Route.ClientLoaderArgs) => {
  const searchParams = new URL(request.url).searchParams;
  const query = searchParams.toString();
  throw redirect(`/${params.workspaceSlug}/profile/${params.userId}/assigned${query ? `?${query}` : ""}`);
};

export default function ProfileIndex() {
  return null;
}
