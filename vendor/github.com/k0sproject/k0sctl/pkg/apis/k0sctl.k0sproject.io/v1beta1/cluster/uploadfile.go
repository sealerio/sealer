package cluster

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	log "github.com/sirupsen/logrus"
)

type LocalFile struct {
	Path     string
	PermMode string
}

// UploadFile describes a file to be uploaded for the host
type UploadFile struct {
	Name            string       `yaml:"name,omitempty"`
	Source          string       `yaml:"src"`
	DestinationDir  string       `yaml:"dstDir"`
	DestinationFile string       `yaml:"dst"`
	PermMode        interface{}  `yaml:"perm"`
	DirPermMode     interface{}  `yaml:"dirPerm"`
	User            string       `yaml:"user"`
	Group           string       `yaml:"group"`
	PermString      string       `yaml:"-"`
	DirPermString   string       `yaml:"-"`
	Sources         []*LocalFile `yaml:"-"`
	Base            string       `yaml:"-"`
}

func (u UploadFile) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Source, validation.Required),
		validation.Field(&u.DestinationFile, validation.Required.When(u.DestinationDir == "").Error("dst or dstdir required")),
		validation.Field(&u.DestinationDir, validation.Required.When(u.DestinationFile == "").Error("dst or dstdir required")),
	)
}

// converts string or integer value to octal string for chmod
func permToString(val interface{}) (string, error) {
	var s string
	switch t := val.(type) {
	case int, float64:
		var num int
		if n, ok := t.(float64); ok {
			num = int(n)
		} else {
			num = t.(int)
		}

		if num < 0 {
			return s, fmt.Errorf("invalid permission: %d: must be a positive value", num)
		}
		if num == 0 {
			return s, fmt.Errorf("invalid nil permission")
		}
		s = fmt.Sprintf("%#o", num)
	case string:
		s = t
	default:
		return "", nil
	}

	for i, c := range s {
		n, err := strconv.Atoi(string(c))
		if err != nil {
			return s, fmt.Errorf("failed to parse permission %s: %w", s, err)
		}

		// These could catch some weird octal conversion mistakes
		if i == 1 && n < 4 {
			return s, fmt.Errorf("invalid permission %s: owner would have unconventional access", s)
		}
		if n > 7 {
			return s, fmt.Errorf("invalid permission %s: octal value can't have numbers over 7", s)
		}
	}

	return s, nil
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (u *UploadFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type uploadFile UploadFile
	yu := (*uploadFile)(u)

	if err := unmarshal(yu); err != nil {
		return err
	}

	fp, err := permToString(u.PermMode)
	if err != nil {
		return err
	}
	u.PermString = fp

	dp, err := permToString(u.DirPermMode)
	if err != nil {
		return err
	}
	u.DirPermString = dp

	return u.resolve()
}

// String returns the file bundle name or if it is empty, the source.
func (u *UploadFile) String() string {
	if u.Name == "" {
		return u.Source
	}
	return u.Name
}

// Owner returns a chown compatible user:group string from User and Group, or empty when neither are set.
func (u *UploadFile) Owner() string {
	return strings.TrimSuffix(fmt.Sprintf("%s:%s", u.User, u.Group), ":")
}

// returns true if the string contains any glob characters
func isGlob(s string) bool {
	return strings.ContainsAny(s, "*%?[]{}")
}

// sets the destination and resolves any globs/local paths into u.Sources
func (u *UploadFile) resolve() error {
	if u.IsURL() {
		if u.DestinationFile == "" {
			if u.DestinationDir != "" {
				u.DestinationFile = path.Join(u.DestinationDir, path.Base(u.Source))
			} else {
				u.DestinationFile = path.Base(u.Source)
			}
		}
		return nil
	}

	if isGlob(u.Source) {
		return u.glob(u.Source)
	}

	stat, err := os.Stat(u.Source)
	if err != nil {
		return fmt.Errorf("failed to stat local path for %s: %w", u, err)
	}

	if stat.IsDir() {
		log.Tracef("source %s is a directory, assuming %s/**/*", u.Source, u.Source)
		return u.glob(path.Join(u.Source, "**/*"))
	}

	perm := u.PermString
	if perm == "" {
		perm = fmt.Sprintf("%o", stat.Mode())
	}
	u.Base = path.Dir(u.Source)
	u.Sources = []*LocalFile{
		{Path: path.Base(u.Source), PermMode: perm},
	}

	return nil
}

// finds files based on a glob pattern
func (u *UploadFile) glob(src string) error {
	base, pattern := doublestar.SplitPattern(src)
	u.Base = base
	fsys := os.DirFS(base)
	sources, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return err
	}

	for _, s := range sources {
		abs := path.Join(base, s)
		log.Tracef("glob %s found: %s", abs, s)
		stat, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", u, err)
		}

		if stat.IsDir() {
			log.Tracef("%s is a directory", abs)
			continue
		}

		perm := u.PermString
		if perm == "" {
			perm = fmt.Sprintf("%o", stat.Mode())
		}

		u.Sources = append(u.Sources, &LocalFile{Path: s, PermMode: perm})
	}

	if len(u.Sources) == 0 {
		return fmt.Errorf("no files found for %s", u)
	}

	if u.DestinationFile != "" && len(u.Sources) > 1 {
		return fmt.Errorf("found multiple files for %s but single file dst %s defined", u, u.DestinationFile)
	}

	return nil
}

// IsURL returns true if the source is a URL
func (u *UploadFile) IsURL() bool {
	return strings.Contains(u.Source, "://")
}
