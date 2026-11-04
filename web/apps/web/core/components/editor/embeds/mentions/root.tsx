/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// local imports
import { EditorUserMention } from "./user";
import type { TCallbackMentionComponentProps } from "@nerve/editor";

export function EditorMentionsRoot(
  props: TCallbackMentionComponentProps & { workspaceID: string; workspaceSlug: string; projectId: string | undefined }
) {
  const { entity_identifier, entity_name } = props;

  switch (entity_name) {
    case "user_mention":
      return (
        <EditorUserMention
          id={entity_identifier}
          workspaceID={props.workspaceID}
          workspaceSlug={props.workspaceSlug}
          projectId={props.projectId}
        />
      );
    default:
      return null;
  }
}
