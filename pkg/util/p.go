package util

import (
	"github.com/magefile/mage/sh"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	// 检查并下载protoc-gen-go和protoc-gen-go-grpc插件
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		sh.Run("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@latest")
	}
	if _, err := exec.LookPath("protoc-gen-go-grpc"); err != nil {
		sh.Run("go", "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest")
	}
}

// Compile compiles the protobuf files
func Compile() error {
	protoPath := "./pkg/protocol"
	dirs, err := os.ReadDir(protoPath)
	if err != nil {
		return err
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
			if err := sh.Run("protoc", args...); err != nil {
				return err
			}

			// Replace "omitempty" in *.pb.go files
			files, _ := filepath.Glob(filepath.Join(outputDir, "*.pb.go"))
			for _, file := range files {
				if err := sh.Run("sed", "-i", "", `s/,omitempty\"/\"/g`, file); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
