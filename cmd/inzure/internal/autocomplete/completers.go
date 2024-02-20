package autocomplete

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ivision-research/inzure/pkg/inzure"
)

func TargetsAutoComplete(inc string, args []string) Completions {
	comps := make(Completions, 0, 5)
	for k, v := range inzure.AvailableTargets {
		if v == 0 {
			continue
		}
		if strings.HasPrefix(k, inc) {
			comps = append(comps, Completion{
				Completion: k,
			})
		}
	}
	return comps
}

func InzureJSONAutoComplete(thisFlag string) func(string, []string) Completions {
	return func(inc string, args []string) Completions {
		return filterDir(inc, func(info os.FileInfo) (string, bool) {
			fileName := info.Name()
			if info.IsDir() {
				return fileName + string(os.PathSeparator), true
			}
			ext := filepath.Ext(fileName)
			if ext == ".json" || ext == inzure.EncryptedFileExtension {

				// Prevent duplicates for the same flag
				if thisFlag != "" {
					sawThisFlag := false
					for _, arg := range args {
						if sawThisFlag {
							if arg == fileName {
								return "", false
							} else {
								sawThisFlag = false
							}
						} else {
							sawThisFlag = arg == thisFlag
						}
					}
				}

				return fileName, true
			}
			return "", false
		})
	}
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
			if inc == "" || strings.HasPrefix(name, inc) || strings.Contains(name, inc) {
				c = append(c, Completion{Completion: name})
			}
		}
	}
	return c
}

func FileAutoComplete(inc string, _ []string) Completions {
	return filterDir(inc, func(info os.FileInfo) (string, bool) {
		if info.IsDir() {
			return info.Name() + string(os.PathSeparator), true
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
