package components

import (
	"strconv"

	"github.com/xmhrnx/giode/styles/properties"
)

func giodeWidth(dp int) properties.Width { return properties.Width(dp) }

func formatInt(n int64) string { return strconv.FormatInt(n, 10) }
