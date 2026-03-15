package selfupdate

import "testing"

func TestGetAdditionalArchAmd64(t *testing.T) {
	archs := getAdditionalArch("amd64", 0, "")
	if len(archs) != 2 || archs[0] != "amd64" || archs[1] != "x86_64" {
		t.Errorf("unexpected archs for amd64: %v", archs)
	}
}

func TestGetAdditionalArchArm(t *testing.T) {
	archs := getAdditionalArch("arm", 7, "")
	expected := []string{"armv7", "armv6", "armv5", "arm"}
	if len(archs) != len(expected) {
		t.Fatalf("expected %d archs, got %d: %v", len(expected), len(archs), archs)
	}
	for i, e := range expected {
		if archs[i] != e {
			t.Errorf("archs[%d] = %q, want %q", i, archs[i], e)
		}
	}
}

func TestGetAdditionalArchArm5(t *testing.T) {
	archs := getAdditionalArch("arm", 5, "")
	if len(archs) != 2 || archs[0] != "armv5" || archs[1] != "arm" {
		t.Errorf("unexpected archs for arm5: %v", archs)
	}
}

func TestGetAdditionalArchArmNoVersion(t *testing.T) {
	archs := getAdditionalArch("arm", 0, "")
	if len(archs) != 1 || archs[0] != "arm" {
		t.Errorf("unexpected archs: %v", archs)
	}
}

func TestGetAdditionalArchArm64(t *testing.T) {
	archs := getAdditionalArch("arm64", 0, "")
	if len(archs) != 1 || archs[0] != "arm64" {
		t.Errorf("unexpected archs: %v", archs)
	}
}

func TestGetAdditionalArchUniversal(t *testing.T) {
	archs := getAdditionalArch("amd64", 0, "universal")
	if len(archs) != 3 || archs[2] != "universal" {
		t.Errorf("unexpected archs with universal: %v", archs)
	}
}

func TestGetAdditionalArchArmOutOfRange(t *testing.T) {
	archs := getAdditionalArch("arm", 8, "")
	if len(archs) != 1 || archs[0] != "arm" {
		t.Errorf("unexpected archs for arm8: %v", archs)
	}
}
