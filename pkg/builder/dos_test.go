package builder

import "testing"

func TestDosDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\PRINCE\\SETUP.EXE", "PRINCE"},
		{"C:\\PRINCE\\SUBDIR\\SETUP.EXE", "SUBDIR"},
		{"C:\\SETUP.EXE", ""},
		{"C:", ""},
		{"C:\\", ""},
		{"/unix/path", ""},
		{"PRINCE/SETUP.EXE", "PRINCE"},
		{"SETUP.EXE", ""},
	}

	for _, tt := range tests {
		result := dosDir(tt.input)
		if result != tt.expected {
			t.Errorf("dosDir(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDosBase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\PRINCE\\SETUP.EXE", "SETUP.EXE"},
		{"C:\\PRINCE\\SUBDIR\\SETUP.EXE", "SETUP.EXE"},
		{"C:\\SETUP.EXE", "SETUP.EXE"},
		{"C:", "C:"},
		{"/unix/path/file.exe", "file.exe"},
		{"SETUP.EXE", "SETUP.EXE"},
	}

	for _, tt := range tests {
		result := dosBase(tt.input)
		if result != tt.expected {
			t.Errorf("dosBase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDosPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/unix/path", "\\unix\\path"},
		{"PRINCE/SETUP.EXE", "PRINCE\\SETUP.EXE"},
		{"C:\\WINDOWS", "C:\\WINDOWS"},
	}

	for _, tt := range tests {
		result := dosPath(tt.input)
		if result != tt.expected {
			t.Errorf("dosPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestInstallDirFromCmd(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\PRINCE\\SETUP.EXE", "PRINCE"},
		{"C:\\GAMES\\INSTALL\\SETUP.EXE", "INSTALL"},
	}

	for _, tt := range tests {
		result := installDirFromCmd(tt.input)
		if result != tt.expected {
			t.Errorf("installDirFromCmd(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestUnixToDosPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/unix/path", "\\unix\\path"},
		{"PRINCE/SETUP.EXE", "PRINCE\\SETUP.EXE"},
		{"C:\\WINDOWS", "C:\\WINDOWS"},
	}

	for _, tt := range tests {
		result := unixToDosPath(tt.input)
		if result != tt.expected {
			t.Errorf("unixToDosPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDosToUnixPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\WINDOWS", "C:/WINDOWS"},
		{"PRINCE\\SETUP.EXE", "PRINCE/SETUP.EXE"},
		{"/unix/path", "/unix/path"},
	}

	for _, tt := range tests {
		result := dosToUnixPath(tt.input)
		if result != tt.expected {
			t.Errorf("dosToUnixPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDosName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\PRINCE", "PRINCE"},
		{"C:\\PRINCE\\", "PRINCE"},
		{"C:\\PRINCE\\SETUP.EXE", "SETUP.EXE"},
		{"C:/PRINCE/", "PRINCE"},
		{"C:/PRINCE/SETUP.EXE", "SETUP.EXE"},
		{"PRINCE", "PRINCE"},
		{"PRINCE\\SETUP.EXE", "SETUP.EXE"},
		{"C:", ""},
		{"C:\\", ""},
	}

	for _, tt := range tests {
		result := dosName(tt.input)
		if result != tt.expected {
			t.Errorf("dosName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
