// Package categorize sorts files into categories by name and MIME type.
package categorize

import (
	"path"
	"strings"

	"github.com/godownload/godownload/internal/model"
)

var byExt = map[string]model.Category{}

func init() {
	add := func(c model.Category, exts string) {
		for _, e := range strings.Fields(exts) {
			byExt[e] = c
		}
	}
	add(model.CategoryVideo, "mp4 mkv avi mov wmv flv webm m4v mpg mpeg 3gp ts m2ts vob ogv rmvb")
	add(model.CategoryMusic, "mp3 flac wav aac m4a ogg opus wma alac aiff ape mka")
	add(model.CategoryDocuments, "pdf doc docx xls xlsx ppt pptx odt ods odp txt rtf epub mobi azw3 csv md djvu")
	add(model.CategoryArchives, "zip rar 7z tar gz tgz bz2 xz zst lz lzma cab iso img")
	add(model.CategoryPrograms, "exe msi msix appx dmg pkg deb rpm appimage apk flatpak snap run bin jar")
	add(model.CategoryImages, "jpg jpeg png gif webp bmp tiff tif svg heic heif avif raw cr2 nef psd ico")
}

// ByName picks the category for a file name, falling back to the MIME type.
func ByName(name, mime string) model.Category {
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(name)), ".")
	if c, ok := byExt[ext]; ok {
		return c
	}
	return byMIME(mime)
}

func byMIME(mime string) model.Category {
	mime = strings.ToLower(mime)
	switch {
	case strings.HasPrefix(mime, "video/"):
		return model.CategoryVideo
	case strings.HasPrefix(mime, "audio/"):
		return model.CategoryMusic
	case strings.HasPrefix(mime, "image/"):
		return model.CategoryImages
	case strings.HasPrefix(mime, "text/"), strings.Contains(mime, "pdf"), strings.Contains(mime, "document"):
		return model.CategoryDocuments
	case strings.Contains(mime, "zip"), strings.Contains(mime, "compressed"), strings.Contains(mime, "x-tar"):
		return model.CategoryArchives
	default:
		return model.CategoryOther
	}
}

// IsMedia reports whether a file can be played while it downloads.
func IsMedia(name string) bool {
	c := ByName(name, "")
	return c == model.CategoryVideo || c == model.CategoryMusic
}

// Largest picks the category for a set of files (a torrent): the category of
// the files that take the most space.
func Largest(names []string, sizes []int64) model.Category {
	total := map[model.Category]int64{}
	for i, n := range names {
		total[ByName(n, "")] += sizes[i]
	}
	best, bestSize := model.CategoryOther, int64(-1)
	for _, c := range model.Categories {
		if total[c] > bestSize {
			best, bestSize = c, total[c]
		}
	}
	return best
}
