package util

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func init() {
	ensureToolsInstalled()
}

func ensureToolsInstalled() {
	tools := map[string]string{
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

	if _, err := exec.LookPath("protoc"); err == nil {
		fmt.Println("protoc is already installed.")
		return
	}
	fmt.Println("Installing protoc...")
	if err := installProtoc(); err != nil {
		fmt.Printf("Failed to install protoc: %s\n", err)
		os.Exit(1)
	}
}

//https://github.com/protocolbuffers/protobuf/releases/download/v26.1/protoc-26.1-linux-x86_64.zip

func installProtoc() error {
	version := "26.1"
	baseURL := "https://github.com/protocolbuffers/protobuf/releases/download/v" + version
	osArch := runtime.GOOS + "-" + runtime.GOARCH
	fileName := fmt.Sprintf("protoc-%s-%s.zip", version, osArch)
	url := baseURL + "/" + fileName

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "protoc-*.zip")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}

	// Unzip the file to /usr/local/bin (you might want to change this based on your OS)
	// This requires admin privileges, consider where to unzip based on your user privileges
	if err := sh.Run("unzip", tmpFile.Name(), "-d", "/usr/local/bin"); err != nil {
		return err
	}

	return nil
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
