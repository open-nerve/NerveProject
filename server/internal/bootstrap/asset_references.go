package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

type imageReferenceDetacher interface {
	DetachAssetReferences(context.Context, []uuid.UUID) error
}
type imageReferenceDeclaration struct{ Schema, Table, Constraint, Column, Parent string }
type imageReferenceStep struct {
	owner      string
	references []imageReferenceDeclaration
	detach     imageReferenceDetacher
}

// The same fixed steps drive execution and the catalog proof of their order.
type assetImageReferences struct {
	identity  identity.ImageReferences
	workspace workspace.ImageReferences
	project   project.ImageReferences
}

func (r assetImageReferences) steps() []imageReferenceStep {
	steps := []imageReferenceStep{{owner: "identity", detach: r.identity}, {owner: "workspace", detach: r.workspace}, {owner: "project", detach: r.project}}
	for _, v := range r.identity.References() {
		steps[0].references = append(steps[0].references, imageReferenceDeclaration(v))
	}
	for _, v := range r.workspace.References() {
		steps[1].references = append(steps[1].references, imageReferenceDeclaration(v))
	}
	for _, v := range r.project.References() {
		steps[2].references = append(steps[2].references, imageReferenceDeclaration(v))
	}
	return steps
}
func (r assetImageReferences) DetachAssetReferences(ctx context.Context, ids []uuid.UUID) error {
	for _, step := range r.steps() {
		if err := step.detach.DetachAssetReferences(ctx, ids); err != nil {
			return err
		}
	}
	return nil
}
