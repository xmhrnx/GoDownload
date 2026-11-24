package theme

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
)

// CategoryIcon is the icon for a category.
func CategoryIcon(c model.Category) string {
	switch c {
	case model.CategoryVideo:
		return "video"
	case model.CategoryMusic:
		return "music"
	case model.CategoryDocuments:
		return "file-text"
	case model.CategoryArchives:
		return "archive"
	case model.CategoryPrograms:
		return "package"
	case model.CategoryImages:
		return "image"
	default:
		return "file"
	}
}

// CategoryColor tints a category's file badge.
func CategoryColor(c model.Category) giode.Color {
	switch c {
	case model.CategoryVideo:
		return "#A78BFA"
	case model.CategoryMusic:
		return "#F472B6"
	case model.CategoryDocuments:
		return "#60A5FA"
	case model.CategoryArchives:
		return "#FBBF24"
	case model.CategoryPrograms:
		return "#34D399"
	case model.CategoryImages:
		return "#22D3EE"
	default:
		return "#94A3B8"
	}
}

// StatusColor is the color of a status's label and progress bar.
func StatusColor(s model.Status) giode.Color {
	switch s {
	case model.StatusDownloading, model.StatusConnecting:
		return "theme.primary"
	case model.StatusVerifying:
		return "theme.info"
	case model.StatusSeeding:
		return Seed
	case model.StatusCompleted:
		return "theme.success"
	case model.StatusFailed:
		return "theme.danger"
	case model.StatusPaused:
		return "theme.warning"
	default:
		return "theme.muted"
	}
}

// StatusLabel is a status's display text.
func StatusLabel(s model.Status) string {
	switch s {
	case model.StatusConnecting:
		return "Connecting"
	case model.StatusDownloading:
		return "Downloading"
	case model.StatusPaused:
		return "Paused"
	case model.StatusVerifying:
		return "Checking"
	case model.StatusSeeding:
		return "Seeding"
	case model.StatusCompleted:
		return "Completed"
	case model.StatusFailed:
		return "Failed"
	default:
		return "Queued"
	}
}
