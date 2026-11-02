// Package localprotocol defines the local adapter's wire address, shared by
// its signer and HTTP consumer. Storage ports and domain rules do not use it.
package localprotocol

import "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"

const (
	Root         = "/static"
	ObjectPrefix = Root + "/local/"
)

func Path(key domain.ObjectKey) string { return ObjectPrefix + string(key) }
