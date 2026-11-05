// Package mediafilter applies the instance-wide allowed media type setting.
package mediafilter

import (
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/mediatype"
)

// Allowed reports whether media should be processed under the current
// allowlist. An empty allowlist permits every supported media object.
func Allowed(media tg.MessageMediaClass) bool {
	allowed := config.C().Media.AllowedMediaTypes
	if len(allowed) == 0 {
		return true
	}
	typ, ok := mediatype.Classify(media)
	if !ok {
		return false
	}
	return Allows(typ)
}

// Allows reports whether a known media category is enabled.
func Allows(typ mediatype.Type) bool {
	allowed := config.C().Media.AllowedMediaTypes
	if len(allowed) == 0 {
		return true
	}
	for _, value := range allowed {
		if value == string(typ) {
			return true
		}
	}
	return false
}
