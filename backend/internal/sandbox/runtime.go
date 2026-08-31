package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
		exeDir = stripExtendedPrefix(filepath.Dir(exePath))
	}

	if jar := stripExtendedPrefix(os.Getenv("TSUNAGU_SANDBOX_JAR")); jar != "" && fileExists(jar) {
		if javaBin := javaFromEnvOrPath(javaExeName); javaBin != "" {
			return &ResolvedRuntime{JavaBin: javaBin, JarPath: jar, Source: "TSUNAGU_SANDBOX_JAR"}, nil
		}
	}

	if resourceDir := stripExtendedPrefix(os.Getenv("TSUNAGU_RESOURCE_DIR")); resourceDir != "" {
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

	if jarPath := locateSandboxJar(exeDir, jarRelPath); jarPath != "" {
		if javaBin := javaFromEnvOrPath(javaExeName); javaBin != "" {
			return &ResolvedRuntime{JavaBin: javaBin, JarPath: jarPath, Source: "JAVA_HOME/PATH"}, nil
		}
	}

	return nil, fmt.Errorf("no usable Java runtime + sandbox jar found (tried TSUNAGU_SANDBOX_JAR, TSUNAGU_RESOURCE_DIR, bundled dir, JAVA_HOME, PATH)")
}

func javaFromEnvOrPath(javaExeName string) string {
	if javaHome := stripExtendedPrefix(os.Getenv("JAVA_HOME")); javaHome != "" {
		if c := filepath.Join(javaHome, "bin", javaExeName); fileExists(c) {
			return c
		}
	}
	if p, err := exec.LookPath(javaExeName); err == nil {
		return stripExtendedPrefix(p)
	}
	return ""
}

// stripExtendedPrefix removes the Windows extended-length path prefix (\\?\ or
// \\?\UNC\). The HotSpot class loader resolves java.home from argv[0] and cannot
// parse that prefix, which makes a jlink'd runtime abort with
// "jimage file name is null" before main. Harmless on non-Windows.
func stripExtendedPrefix(p string) string {
	if strings.HasPrefix(p, `\\?\UNC\`) {
		return `\\` + p[len(`\\?\UNC\`):]
	}
	if strings.HasPrefix(p, `\\?\`) {
		return p[len(`\\?\`):]
	}
	return p
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
