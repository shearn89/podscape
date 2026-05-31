package floorplan

// Density controls how wide each node card is, which in turn determines how
// many cards fit per row at a given terminal width.
type Density int

const (
	DensityCompact Density = iota
	DensityNormal
	DensityWide
)

// CardWidth returns the inner card width (border included) for a density.
func (d Density) CardWidth() int {
	switch d {
	case DensityCompact:
		return 18
	case DensityWide:
		return 40
	default:
		return 26
	}
}

// PodChipWidth returns how wide a single pod chip is rendered in this density.
func (d Density) PodChipWidth() int {
	switch d {
	case DensityCompact:
		return 3
	case DensityWide:
		return 16
	default:
		return 10
	}
}

func (d Density) String() string {
	switch d {
	case DensityCompact:
		return "compact"
	case DensityWide:
		return "wide"
	default:
		return "normal"
	}
}
