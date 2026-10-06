package events

// Theme editor actions (backend/application/theme) -- extend the
// storefront.theme.* family (created / published / rolled_back / deleted
// are typed events in theme_events.go). Session tokens never go into an
// event.
var (
	ThemeRenamed = defineAction("storefront.theme.renamed",
		"A theme was renamed. changes: name.")
	ThemeDraftStarted = defineAction("storefront.theme.draft_started",
		"Editing a published theme started a new draft version (a copy of the live one). data: version_id, version_number, based_on_version_id.")
	ThemeSettingsUpdated = defineAction("storefront.theme.settings_updated",
		"A theme draft's global settings (colors, fonts, logo...) were saved. data: version_id, revision.")
	ThemePageCreated = defineAction("storefront.theme_page.created",
		"A page was added to a theme draft. data: theme_id, version_id, key, title, revision.")
	ThemePageDeleted = defineAction("storefront.theme_page.deleted",
		"A page was removed from a theme draft. data: theme_id, version_id, key, revision.")
	ThemeDocumentUpdated = defineAction("storefront.theme_document.updated",
		"A theme draft document (page, header, footer...) was saved. data: theme_id, version_id, kind, key, revision.")
	ThemeSessionCreated = defineAction("storefront.theme_session.created",
		"An editor or preview session was opened for a theme. data: theme_id, purpose, user_id, role, locale.")
	ThemeAssetUploaded = defineAction("storefront.theme_asset.uploaded",
		"An image was added to the tenant's theme asset library. data: filename, content_type, bytes, width, height.")
	ThemeAssetUpdated = defineAction("storefront.theme_asset.updated",
		"A theme asset's details changed. changes: alt.")
	ThemeAssetDeleted = defineAction("storefront.theme_asset.deleted",
		"A theme asset was removed from the library. data: filename.")
)
