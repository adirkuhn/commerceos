package channels

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// importImages downloads each Woo image and re-uploads it through our own
// FileStorage rather than just linking Woo's URL — the product stays
// viewable if the store connection is later removed, and we're not
// hotlinking another site's media indefinitely. Runs after the product's
// own transaction has committed (mediaUploader opens its own transactions
// around a storage write — see mediaUploader's doc comment). Per-image
// failures are recorded and never abort the rest of the product's images
// or the run.
func (uc *ImportWooProductsUseCase) importImages(ctx context.Context, client integrationsDomain.WooClient, productID string, wp integrationsDomain.WooProduct, result *ImportResult) {
	for _, img := range wp.Images {
		if img.URL == "" {
			continue
		}
		data, contentType, err := client.DownloadImage(ctx, img.URL)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: image %s: %v", wp.ID, img.URL, err))
			continue
		}
		contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
		_, err = uc.media.Execute(ctx, apppms.UploadMediaInput{
			ProductID: &productID, Filename: filenameFromURL(img.URL), ContentType: contentType, Data: data, AltText: img.Alt,
		})
		data.Close()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: image %s: %v", wp.ID, img.URL, err))
		}
	}
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "image"
	}
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "image"
	}
	return name
}
