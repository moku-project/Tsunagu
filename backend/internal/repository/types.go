package repository

type ParsedExtension struct {
	Name        string
	PackageName string
	ApkURL      string
	JarURL      string
	IconURL     string
	VersionName string
	Lang        string
	ContentType string
}

type repoIndex struct {
	Name            string
	BadgeLabel      string
	SigningKey      string
	Contact         contact
	ExtensionList   *extensionList
	ExtensionListURL string
}

type contact struct {
	Website string
	Discord string
}

type extensionList struct {
	Extensions []extension
}

type extension struct {
	Name          string
	PackageName   string
	Resources     resources
	ExtensionLib  string
	VersionCode   int64
	VersionName   string
	ContentWarning int32
	Sources       []source
}

type resources struct {
	ApkURL  string
	IconURL string
	JarURL  string
}

type source struct {
	ID         int64
	Name       string
	Language   string
	HomeURL    string
	MirrorURLs []string
	Message    string
}

type legacyRepoExtension struct {
	Name    string `json:"name"`
	Pkg     string `json:"pkg"`
	Apk     string `json:"apk"`
	Lang    string `json:"lang"`
	Version string `json:"version"`
}

type novelRepoEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Site    string `json:"site"`
	Lang    string `json:"lang"`
	Version string `json:"version"`
	URL     string `json:"url"`
	IconURL string `json:"iconUrl"`
}

func (idx *repoIndex) toParsedExtensions(baseRawURL string) []ParsedExtension {
	if idx.ExtensionList == nil {
		return nil
	}
	out := make([]ParsedExtension, 0, len(idx.ExtensionList.Extensions))
	for _, ext := range idx.ExtensionList.Extensions {
		langs := map[string]struct{}{}
		for _, s := range ext.Sources {
			langs[s.Language] = struct{}{}
		}
		lang := "all"
		if len(langs) == 1 {
			for l := range langs {
				lang = l
			}
		}
		out = append(out, ParsedExtension{
			Name:        ext.Name,
			PackageName: ext.PackageName,
			ApkURL:      ext.Resources.ApkURL,
			JarURL:      ext.Resources.JarURL,
			IconURL:     ext.Resources.IconURL,
			VersionName: ext.VersionName,
			Lang:        lang,
		})
	}
	return out
}

func legacyToParsedExtensions(exts []legacyRepoExtension, apkBaseURL string) []ParsedExtension {
	out := make([]ParsedExtension, 0, len(exts))
	for _, ext := range exts {
		out = append(out, ParsedExtension{
			Name:        ext.Name,
			PackageName: ext.Pkg,
			ApkURL:      apkBaseURL + ext.Apk,
			JarURL:      "",
			IconURL:     "",
			VersionName: ext.Version,
			Lang:        ext.Lang,
		})
	}
	return out
}

func novelToParsedExtensions(exts []novelRepoEntry) []ParsedExtension {
	out := make([]ParsedExtension, 0, len(exts))
	for _, ext := range exts {
		out = append(out, ParsedExtension{
			Name:        ext.Name,
			PackageName: ext.ID,
			ApkURL:      "",
			JarURL:      ext.URL,
			IconURL:     ext.IconURL,
			VersionName: ext.Version,
			Lang:        ext.Lang,
			ContentType: "novel",
		})
	}
	return out
}
