package theme

import (
	_ "embed"

	"github.com/xmhrnx/giode"
)

// Font is the UI typeface (Inter, SIL Open Font License; see fonts/OFL.txt).
const Font = "Inter"

var (
	//go:embed fonts/Inter-Regular.ttf
	interRegular []byte
	//go:embed fonts/Inter-Medium.ttf
	interMedium []byte
	//go:embed fonts/Inter-SemiBold.ttf
	interSemiBold []byte
	//go:embed fonts/Inter-Bold.ttf
	interBold []byte
)

func registerFonts() error {
	// The Go fonts cover symbols Inter lacks.
	giode.UseGofont()
	return giode.RegisterFont(
		giode.FontFile{Family: Font, Data: interRegular},
		giode.FontFile{Family: Font, Weight: giode.FontWeightMedium, Data: interMedium},
		giode.FontFile{Family: Font, Weight: giode.FontWeightSemiBold, Data: interSemiBold},
		giode.FontFile{Family: Font, Weight: giode.FontWeightBold, Data: interBold},
	)
}
