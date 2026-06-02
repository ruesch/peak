package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/aleksana/peak/internal/vfs/afero"
)

func getVFS() afero.Fs {
	if appEditor != nil && appEditor.ninep != nil {
		return appEditor.ninep.vfs
	}
	return afero.NewOsFs()
}

// toDir ensures a directory path ends with a trailing slash.
func toDir(path string) string {
	if path != "" && !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// getPathDir returns the directory associated with a path.
func getPathDir(path string) string {
	if path == "" {
		return getwd()
	}
	if strings.HasSuffix(path, "/") {
		return path
	}
	return toDir(filepath.Dir(resolvePath(path)))
}

// resolvePath returns an absolute path, expanding ~ and handling relative segments.
func resolvePath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[1:])
		}
	}
	abs, _ := filepath.Abs(path)
	return abs
}

// resolveWithContext resolves a path within a given context directory.
func resolveWithContext(path, contextDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "~") {
		return resolvePath(path)
	}
	if contextDir == "" {
		contextDir = getwd()
	}
	return filepath.Join(contextDir, path)
}

// formatPath formats a full path relative to a context path.
func formatPath(fullPath, contextPath string) string {
	if contextPath == "" {
		return fullPath
	}

	if strings.HasPrefix(contextPath, "~") {
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(fullPath, home) {
			return "~" + fullPath[len(home):]
		}
	} else if !filepath.IsAbs(contextPath) {
		cwd, _ := os.Getwd()
		if rel, err := filepath.Rel(cwd, fullPath); err == nil {
			if !strings.HasPrefix(rel, ".") && !strings.HasPrefix(rel, "/") {
				rel = "./" + rel
			}
			return rel
		}
	}
	return fullPath
}

// getwd returns the current working directory with a trailing slash.
func getwd() string {
	dir, _ := os.Getwd()
	return toDir(dir)
}

// readFile reads data from a file.
func readFile(path string) ([]byte, error) {
	return afero.ReadFile(getVFS(), path)
}

// writeFile writes data to a file.
func writeFile(path string, data []byte) error {
	return afero.WriteFile(getVFS(), path, data, 0644)
}

// readFileOrDir returns the content of a file or a listing if it's a directory,
// and whether the file is writable (owner-write permission bit set).
func readFileOrDir(path string) (string, bool, bool, error) {
	fi, err := getVFS().Stat(path)
	if err != nil {
		return "", false, false, err
	}
	writable := !fi.IsDir() && fi.Mode().Perm()&0200 != 0
	if fi.IsDir() {
		content, err := listDir(path)
		return content, true, writable, err
	}

	f, err := getVFS().Open(path)
	if err != nil {
		return "", false, writable, err
	}
	defer f.Close()

	size := fi.Size()
	if size > 0 {
		data := make([]byte, size)
		n, err := f.ReadAt(data, 0)
		if err != nil && err != io.EOF {
			return "", false, writable, err
		}
		if int64(n) < int64(len(data)) || err == io.EOF {
			if err := isBinary(data[:n]); err != nil {
				return "", false, writable, err
			}
			return string(data[:n]), false, writable, nil
		}
		if err := isBinary(data); err != nil {
			return "", false, writable, err
		}
		content, err := readFileTail(f, data, int64(len(data)))
		return content, false, writable, err
	}
	content, err := readFileTail(f, nil, 0)
	return content, false, writable, err
}

func isBinary(data []byte) error {
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return fmt.Errorf("binary file")
		}
	}
	return nil
}

func readFileTail(f afero.File, prefix []byte, off int64) (string, error) {
	chunks := prefix
	buf := make([]byte, 4096)
	for {
		n, err := f.ReadAt(buf, off)
		if n > 0 {
			if len(chunks) == 0 {
				if err := isBinary(buf[:n]); err != nil {
					return "", err
				}
			}
			chunks = append(chunks, buf[:n]...)
			off += int64(n)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return string(chunks), nil
}

// listDir returns a formatted string listing the contents of a directory.
func listDir(path string) (string, error) {
	entries, err := afero.ReadDir(getVFS(), path)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var sb strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		sb.WriteString(name + "\n")
	}
	return sb.String(), nil
}

func join(elem ...string) string {
	return filepath.Join(elem...)
}

// runCommand runs a command with sh -c and returns the output and error.
func runCommand(cmd, path, input string, winid int) (string, error) {
	if appEditor != nil && appEditor.ninep != nil {
		ninep := appEditor.ninep
		dir := getPathDir(path)
		if mountPath, mountFs := ninep.FindMount(dir); mountPath != "" {
			relPath, _ := filepath.Rel(mountPath, dir)
			if runF, err := mountFs.OpenFile("run", os.O_RDWR, 0); err == nil {
				out, rerr := remoteRun(runF, relPath+"/", cmd)
				runF.Close()
				return out, rerr
			}
		}
		if localDir, ok := ninep.ResolveLocalPath(dir); ok {
			return runLocalCommand(cmd, path, localDir, input, winid)
		}
		return "", fmt.Errorf("%s: don't know how to run command", path)
	}
	return runLocalCommand(cmd, path, getPathDir(path), input, winid)
}


func remoteRun(f afero.File, relPath, cmd string) (string, error) {
	if _, err := f.WriteAt([]byte(relPath+"\n"+cmd+"\n"), 0); err != nil {
		return "", err
	}
	var sb strings.Builder
	buf := make([]byte, 4096)
	var off int64
	for {
		n, err := f.ReadAt(buf, off)
		if n > 0 {
			sb.Write(buf[:n])
			off += int64(n)
		}
		if err != nil {
			break
		}
	}
	return sb.String(), nil
}

// runLocalCommand executes a command on the local OS.
func runLocalCommand(cmd, path, dir, input string, winid int) (string, error) {
	wrappedCmd := fmt.Sprintf("env samfile=%s winid=%d sh -c %s",
		shellescape.Quote(path),
		winid,
		shellescape.Quote(cmd))

	c := exec.Command("sh", "-c", wrappedCmd)
	c.Dir = dir
	if input != "" {
		c.Stdin = strings.NewReader(input)
	}

	out, err := c.CombinedOutput()
	return string(out), err
}
