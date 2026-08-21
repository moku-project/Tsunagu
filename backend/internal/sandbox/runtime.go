package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type ResolvedRuntime struct {
	JavaBin string
	JarPath string
	Source  string
}

func resolveRuntime(jarRelPath string) (*ResolvedRuntime, error) {
	javaExeName := "java"
	if runtime.GOOS == "windows" {
		javaExeName = "java.exe"
	}
	var exeDir string
	if exePath, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exePath)
	}

	if resourceDir := os.Getenv("TSUNAGU_RESOURCE_DIR"); resourceDir != "" {
		javaBin := filepath.Join(resourceDir, "sandbox", "runtime", "bin", javaExeName)
		jarPath := filepath.Join(resourceDir, "sandbox", jarRelPath)
		if fileExists(javaBin) && fileExists(jarPath) {
			return &ResolvedRuntime{JavaBin: javaBin, JarPath: jarPath, Source: "TSUNAGU_RESOURCE_DIR"}, nil
		}
	}

	if exeDir != "" {
		javaBin := filepath.Join(exeDir, "sandbox", "runtime", "bin", javaExeName)
		jarPath := filepath.Join(exeDir, "sandbox", jarRelPath)
		if fileExists(javaBin) && fileExists(jarPath) {
			return &ResolvedRuntime{JavaBin: javaBin, JarPath: jarPath, Source: "bundled"}, nil
		}
	}

	jarPath := locateSandboxJar(exeDir, jarRelPath)

	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		javaBin := filepath.Join(javaHome, "bin", javaExeName)
		if fileExists(javaBin) && jarPath != "" {
			return &ResolvedRuntime{JavaBin: javaBin, JarPath: jarPath, Source: "JAVA_HOME"}, nil
		}
	}

	if p, err := exec.LookPath(javaExeName); err == nil {
		if jarPath != "" {
			return &ResolvedRuntime{JavaBin: p, JarPath: jarPath, Source: "PATH"}, nil
		}
	}

	return nil, fmt.Errorf("no usable Java runtime + sandbox jar found (tried TSUNAGU_RESOURCE_DIR, bundled dir, JAVA_HOME, PATH)")
}

func locateSandboxJar(exeDir, jarRelPath string) string {
	if exeDir != "" {
		p := filepath.Join(exeDir, "sandbox", jarRelPath)
		if fileExists(p) {
			return p
		}
	}

	if repoRoot := findRepoRoot(); repoRoot != "" {
		matches, err := filepath.Glob(filepath.Join(repoRoot, "sandbox", "build", "libs", "sandbox-*-all.jar"))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

func findRepoRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
