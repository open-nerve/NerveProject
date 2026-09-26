// Package domain holds the instance module's model.
package domain

// Product is the product name every instance reports.
const Product = "Nerve"

// APIVersion is the version of the HTTP API this build serves. It follows
// the product's major version (v0 design 3.1) and prefixes every API path.
const APIVersion = "v0"

// Build identifies the binary an instance runs.
type Build struct {
	Version string // product version, e.g. "0.1.0-dev"
	Commit  string // git revision, "unknown" without a VCS stamp
}

// Settings are the configuration that an instance reports to clients, so
// that they adapt to it (M2 design 5.3). The modules that act on them
// enforce them.
type Settings struct {
	SignupEnabled            bool  // auth.signup_enabled
	WorkspaceCreationEnabled bool  // workspace.creation_enabled
	FileSizeLimit            int64 // files.size_limit, in bytes
}

// Info is what an instance tells API clients about itself.
type Info struct {
	Product    string
	Version    string
	Commit     string
	APIVersion string
	Settings
}
