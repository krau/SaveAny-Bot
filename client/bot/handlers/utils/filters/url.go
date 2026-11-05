package filters

import (
	"regexp"
	"slices"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/mygotg/dispatcher/handlers/filters"
	"github.com/krau/mygotg/types"
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
