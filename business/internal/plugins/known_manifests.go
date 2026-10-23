package plugins

// EmbeddedSlugs devolve os slugs dos plugins embarcados no binário, pelo manifesto local de cada um. É a lista de
// plugins que uma empresa pode ativar (não há catálogo externo).
func EmbeddedSlugs() []string {
	return []string{
		(&HelpdeskPlugin{}).GetManifest().Slug,
		(&WebchatPlugin{}).GetManifest().Slug,
		(&AssistantPlugin{}).GetManifest().Slug,
		(&GroupsPlugin{}).GetManifest().Slug,
		(&InventoryAdvancedPlugin{}).GetManifest().Slug,
	}
}

// IsEmbeddedSlug diz se o slug é de um plugin embarcado.
func IsEmbeddedSlug(slug string) bool {
	for _, s := range EmbeddedSlugs() {
		if s == slug {
			return true
		}
	}
	return false
}
