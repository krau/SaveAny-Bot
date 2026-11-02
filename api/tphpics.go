package api

import (
	"context"

	"github.com/krau/SaveAny-Bot/common/utils/tphutil"
)

// ExtractTelegraphImages fetches image URLs from a Telegraph page. The shared
// helper keeps the API and bot paths on the same URL validation and client.
func ExtractTelegraphImages(ctx context.Context, pageURL string) ([]string, string, error) {
	return tphutil.ExtractImages(ctx, pageURL)
}
