package autocomplete

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/CarveSystems/inzure"
)

func InzureJSONAutoComplete(inc string, args []string) Completions {
	files := FileAutoComplete(inc, args)
	filtered := make(Completions, 0, 5)
	for _, f := range files {
		ext := filepath.Ext(f.Completion)
		if ext == ".json" || ext == inzure.EncryptedFileExtension {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func DirAutoComplete(inc string, _ []string) Completions {
	return filterDir(inc, func(info os.FileInfo) (string, bool) {
		if !info.IsDir() {
			return "", false
		}
		return info.Name(), true
	})
}

func filterDir(inc string, f func(os.FileInfo) (string, bool)) Completions {
	c := make(Completions, 0, 5)
	dir := inc
	if dir == "" {
		dir = "."
	}
	s, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			dir = filepath.Dir(dir)
			s, err = os.Stat(dir)
			if err != nil {
				return c
			}
		} else {
			return c
		}
	}
	if !s.IsDir() {
		return c
	}

	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return c
	}
	for _, info := range infos {
		name, use := f(info)
		if use {
			name = filepath.Join(dir, name)
			if strings.HasPrefix(name, inc) {
				c = append(c, Completion{Completion: name})
			}
		}
	}
	return c
}

func FileAutoComplete(inc string, _ []string) Completions {
	return filterDir(inc, func(info os.FileInfo) (string, bool) {
		if info.IsDir() {
			return filepath.Join(info.Name(), ""), true
		}
		return info.Name(), true
	})
}

func IQSAutoComplete(inc string, args []string) Completions {
	l := len(inc)
	idx := 0
	for idx < l {
		if inc[idx] == '\'' || inc[idx] == '"' {
			idx++
		} else {
			break
		}
	}
	inc = inc[idx:]
	if !strings.HasPrefix(inc, "/") {
		inc = "/" + inc
	}
	p := parsePartialQueryString(inc)
	switch p.Parts {
	case 1:
		return p.rgFieldAutoComplete()
	case 2:
		return p.rgAutoComplete(args)
	case 3:
		return p.nameAutoComplete(args)
	default:
	}
	return Completions([]Completion{})
}
