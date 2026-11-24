package model

// Category groups downloads by file type in the sidebar and picks their folder.
type Category string

const (
	CategoryVideo     Category = "video"
	CategoryMusic     Category = "music"
	CategoryDocuments Category = "documents"
	CategoryArchives  Category = "archives"
	CategoryPrograms  Category = "programs"
	CategoryImages    Category = "images"
	CategoryOther     Category = "other"
)

// Categories lists every category in sidebar order.
var Categories = []Category{
	CategoryVideo, CategoryMusic, CategoryDocuments, CategoryArchives,
	CategoryPrograms, CategoryImages, CategoryOther,
}

// Title is the category's display name, which is also its default folder name.
func (c Category) Title() string {
	switch c {
	case CategoryVideo:
		return "Video"
	case CategoryMusic:
		return "Music"
	case CategoryDocuments:
		return "Documents"
	case CategoryArchives:
		return "Compressed"
	case CategoryPrograms:
		return "Programs"
	case CategoryImages:
		return "Images"
	default:
		return "Other"
	}
}
