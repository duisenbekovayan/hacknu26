package envfile

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Load ищет вверх от рабочей директории каталог с `.env` / `.env.local` / `llm.env`.
// Порядок: сначала `.env`, затем `.env.local` и `llm.env` (переопределяют уже заданные ключи).
func Load() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	for d := wd; ; {
		envPath := filepath.Join(d, ".env")
		localPath := filepath.Join(d, ".env.local")
		llmPath := filepath.Join(d, "llm.env")
		var has bool
		for _, p := range []string{envPath, localPath, llmPath} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				has = true
				break
			}
		}
		if has {
			if st, err := os.Stat(envPath); err == nil && !st.IsDir() {
				_ = loadFile(envPath, false)
			}
			if st, err := os.Stat(localPath); err == nil && !st.IsDir() {
				_ = loadFile(localPath, true)
			}
			if st, err := os.Stat(llmPath); err == nil && !st.IsDir() {
				_ = loadFile(llmPath, true)
			}
			return
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
}

func loadFile(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// длинные строки (ключи)
	const max = 512 * 1024
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, max)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			key := strings.TrimSpace(line[:i])
			val := strings.TrimSpace(line[i+1:])
			if key == "" {
				continue
			}
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}
			if override || os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
	return sc.Err()
}
