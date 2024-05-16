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

func ensureToolsInstalled() {
	tools := map[string]string{
		"protoc-gen-go":      "google.golang.org/protobuf/cmd/protoc-gen-go",
		"protoc-gen-go-grpc": "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
	}
	targetDir := "/usr/local/bin" // 指定安装目录
	for tool, path := range tools {
		targetPath := filepath.Join(targetDir, tool)
		if _, err := exec.LookPath(targetPath); err != nil {
			fmt.Printf("Building and installing %s...\n", tool)
			if err := goBuildInstall(path, tool, targetDir); err != nil {
				fmt.Printf("Failed to install %s: %s\n", tool, err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("%s is already installed.\n", tool)
		}
	}

	if _, err := exec.LookPath(filepath.Join(targetDir, "protoc")); err == nil {
		fmt.Println("protoc is already installed.")
		return
	}
	fmt.Println("Installing protoc...")
	if err := installProtoc(); err != nil {
		fmt.Printf("Failed to install protoc: %s\n", err)
		os.Exit(1)
	}
}

func goBuildInstall(packagePath, binaryName, installDir string) error {
	cmd := exec.Command("go", "build", "-o", filepath.Join(installDir, binaryName), packagePath)
	cmd.Env = append(os.Environ(), "GOBIN="+installDir)
	return cmd.Run()
}

//https://github.com/protocolbuffers/protobuf/releases/download/v26.1/protoc-26.1-linux-amd64.zip
//https://github.com/protocolbuffers/protobuf/releases/download/v26.1/protoc-26.1-linux-x86_64.zip

func getProtocArch(archMap map[string]string, goArch string) string {
	if arch, ok := archMap[goArch]; ok {
		return arch
	}
	return goArch
}

func installProtoc() error {

	version := "26.1"
	baseURL := "https://github.com/protocolbuffers/protobuf/releases/download/v" + version
	archMap := map[string]string{
		"amd64": "x86_64",
		"386":   "x86",
		"arm64": "aarch64",
	}
	osArch := runtime.GOOS + "-" + getProtocArch(archMap, runtime.GOARCH)
	fileName := fmt.Sprintf("protoc-%s-%s.zip", version, osArch)
	url := baseURL + "/" + fileName

	fmt.Println("URL:", url)

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
	if err := sh.Run("unzip", tmpFile.Name(), "-d", "/usr/local"); err != nil {
		return err
	}

	return nil
}

// Protocol compiles the protobuf files
func Protocol() error {
	ensureToolsInstalled()

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
