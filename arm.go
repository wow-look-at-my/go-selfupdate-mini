package selfupdate

import "debug/buildinfo"

func getGOARM(goBinary string) uint8 {
	build, err := buildinfo.ReadFile(goBinary)
	if err != nil {
		return 0
	}
	for _, setting := range build.Settings {
		if setting.Key == "GOARM" {
			return setting.Value[0] - '0'
		}
	}
	return 0
}
