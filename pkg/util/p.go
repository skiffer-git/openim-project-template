package util

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	//  protoc, protoc-gen-go 和 protoc-gen-go-grpc
	tools := map[string]string{
		"protoc":             "protoc",
		"protoc-gen-go":      "google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		"protoc-gen-go-grpc": "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
	}
	for tool, path := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Printf("Installing %s...\n", tool)
			if err := sh.Run("go", "install", path); err != nil {
				fmt.Printf("Failed to install %s: %s\n", tool, err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("%s is already installed.\n", tool)
		}
	}
}

// Protocol compiles the protobuf files
func Protocol() error {
	protoPath := "./pkg/protocol"
	dirs, err := os.ReadDir(protoPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s", err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			dirName := dir.Name()
			protoFile := filepath.Join(protoPath, dirName, dirName+".proto")
			outputDir := filepath.Join(protoPath, dirName)
			module := "github.com/openimsdk/openim-project-template/pkg/protocol/" + dirName

			args := []string{
				"--go_out=" + outputDir,
				"--go_opt=module=" + module,
				"--go-grpc_out=" + outputDir,
				"--go-grpc_opt=module=" + module,
				protoFile,
			}
			fmt.Printf("Compiling %s...\n", protoFile)
			if err := sh.Run("protoc", args...); err != nil {
				return fmt.Errorf("failed to compile %s: %s", protoFile, err)
			}

			// Replace "omitempty" in *.pb.go files
			files, _ := filepath.Glob(filepath.Join(outputDir, "*.pb.go"))
			for _, file := range files {
				fmt.Printf("Fixing omitempty in %s...\n", file)
				if err := sh.Run("sed", "-i", "", `s/,omitempty\"/\"/g`, file); err != nil {
					return fmt.Errorf("failed to replace omitempty in %s: %s", file, err)
				}
			}
		}
	}
	return nil
}
