package filters

import (
	"regexp"
	"slices"

	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/types"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

func RegexUrl(r *regexp.Regexp) filters.MessageFilter {
	return func(m *types.Message) bool {
		if m.Text == "" {
			return false
		}
		if r.MatchString(m.Text) {
			return true
		}
		urls := tgutil.ExtractMessageEntityUrls(m.Message)
		if len(urls) == 0 {
			return false
		}
		return slices.ContainsFunc(urls, r.MatchString)
	}
}
