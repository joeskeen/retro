package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Instruction struct {
	Type   string
	Name   string
	Args   string
	Source string
	Dest   string
}

type BaseImage struct {
	Name       string
	Constraint string
}

type Retrofile struct {
	BaseImage  BaseImage
	Tag        string
	Copy       []Instruction
	Entrypoint string
	WorkingDir string
	Install    string
}

func ParseRetrofile(path string) (*Retrofile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	rf := &Retrofile{}

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		instr := strings.ToUpper(parts[0])
		var args string
		if len(parts) > 1 {
			args = strings.TrimSpace(parts[1])
		}

		switch instr {
		case "FROM":
			rf.BaseImage = parseBaseImage(args)
		case "TAG":
			rf.Tag = args
		case "COPY":
			copyInstr := parseCopyInstruction(args)
			if copyInstr.Source == "" || copyInstr.Dest == "" {
				return nil, fmt.Errorf("line %d: COPY requires SOURCE and DEST", lineNum+1)
			}
			rf.Copy = append(rf.Copy, copyInstr)
		case "ENTRYPOINT":
			rf.Entrypoint = args
		case "WORKDIR":
			rf.WorkingDir = args
		case "INSTALL":
			rf.Install = args
		default:
			return nil, fmt.Errorf("line %d: unknown instruction %s", lineNum+1, instr)
		}
	}

	if rf.BaseImage.Name == "" {
		return nil, fmt.Errorf("Retrofile requires FROM instruction")
	}

	return rf, nil
}

func parseBaseImage(args string) BaseImage {
	bi := BaseImage{}

	if strings.Contains(args, ":") {
		parts := strings.SplitN(args, ":", 2)
		bi.Name = parts[0]
		bi.Constraint = parts[1]
	} else {
		bi.Name = args
		bi.Constraint = ""
	}

	return bi
}

func parseCopyInstruction(args string) Instruction {
	instr := Instruction{Type: "COPY"}

	parts := strings.Split(args, " ")
	if len(parts) >= 2 {
		instr.Source = parts[0]
		instr.Dest = parts[len(parts)-1]
	}

	instr.Dest = filepath.Clean(instr.Dest)
	return instr
}

func normalizeInstruction(line string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(strings.TrimSpace(line), " ")
}
