package selfupdate

import (
	"testing"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestGetAdditionalArchAmd64(t *testing.T) {
	archs := getAdditionalArch("amd64", 0, "")
	assert.False(t, len(archs) != 2 || archs[0] != "amd64" || archs[1] != "x86_64")

}

func TestGetAdditionalArchArm(t *testing.T) {
	archs := getAdditionalArch("arm", 7, "")
	expected := []string{"armv7", "armv6", "armv5", "arm"}
	require.Equal(t, len(expected), len(archs))

	for i, e := range expected {
		assert.Equal(t, e, archs[i])

	}
}

func TestGetAdditionalArchArm5(t *testing.T) {
	archs := getAdditionalArch("arm", 5, "")
	assert.False(t, len(archs) != 2 || archs[0] != "armv5" || archs[1] != "arm")

}

func TestGetAdditionalArchArmNoVersion(t *testing.T) {
	archs := getAdditionalArch("arm", 0, "")
	assert.False(t, len(archs) != 1 || archs[0] != "arm")

}

func TestGetAdditionalArchArm64(t *testing.T) {
	archs := getAdditionalArch("arm64", 0, "")
	assert.False(t, len(archs) != 1 || archs[0] != "arm64")

}

func TestGetAdditionalArchUniversal(t *testing.T) {
	archs := getAdditionalArch("amd64", 0, "universal")
	assert.False(t, len(archs) != 3 || archs[2] != "universal")

}

func TestGetAdditionalArchArmOutOfRange(t *testing.T) {
	archs := getAdditionalArch("arm", 8, "")
	assert.False(t, len(archs) != 1 || archs[0] != "arm")

}
