// Package asset exposes the file module's composition contract.
package asset

// MAC purposes are stable across process restarts and distinct from every
// other use of the instance key. Payloads additionally bind their own purpose.
const (
	UploadMACPurpose = "asset-upload"
	ReadMACPurpose   = "asset-read"
)
