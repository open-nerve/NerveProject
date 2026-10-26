package domain

// LinkMetadata represents pending previews with an empty object and preview
// fields with nullable strings. Stored keys are title, favicon, url,
// favicon_url and error, constrained by the issue_links schema.
type LinkMetadata map[string]*string
