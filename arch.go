package selfupdate

import "fmt"

const (
	minARM = 5
	maxARM = 7
)

// getAdditionalArch returns candidate architectures to check against assets.
func getAdditionalArch(arch string, goarm uint8, universalArch string) []string {
	additionalArch := make([]string, 0, 3)

	if arch == "arm" && goarm >= minARM && goarm <= maxARM {
		for v := goarm; v >= minARM; v-- {
			additionalArch = append(additionalArch, fmt.Sprintf("armv%d", v))
		}
		additionalArch = append(additionalArch, "arm")
		return additionalArch
	}

	additionalArch = append(additionalArch, arch)
	if arch == "amd64" {
		additionalArch = append(additionalArch, "x86_64")
	}
	if universalArch != "" {
		additionalArch = append(additionalArch, universalArch)
	}
	return additionalArch
}
