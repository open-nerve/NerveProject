/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext } from "react";
import { observer } from "mobx-react";
import { cn } from "@nerve/utils";
import { StoreContext } from "@/lib/store-context";
import { projectCoverURL } from "@/helpers/project-covers";
import { DEFAULT_COVER_IMAGE_URL } from "@/helpers/cover-image.helper";
import { ProfileImage } from "./profile-image";

type Props = { source: { kind: "project"; workspaceID: string; projectID: string }; alt: string; className?: string };
export const ProjectCover = observer(function ProjectCover({ source, alt, className }: Props) {
  const root = useContext(StoreContext);
  const project = root.issue.active ? root.projectRoot.project.getProjectById(source.projectID) : undefined;
  const classes = cn("block bg-layer-2 object-cover", className);
  const empty = <span role={alt ? "img" : undefined} aria-label={alt || undefined} className={classes} />;
  if (!project || project.workspace_id !== source.workspaceID) return empty;
  if (project.cover_builtin_id) {
    const url = projectCoverURL(project.cover_builtin_id);
    return url ? <img src={url} alt={alt} className={classes} /> : empty;
  }
  if (project.cover_image_url) return <ProfileImage source={source} alt={alt} className={classes} />;
  return <img src={DEFAULT_COVER_IMAGE_URL} alt={alt} className={classes} />;
});
