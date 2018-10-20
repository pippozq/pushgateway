package envconf

import (
	"strings"
)

// mask value string for security value
type SecurityStringer interface {
	SecurityString() string
}

func EnvVarFromKeyValue(key string, value string) *EnvVar {
	envVar := &EnvVar{
		KeyPath:    key,
		Value:      value,
		ShouldConf: true,
	}

	keyParts := strings.Split(key, "__")

	if len(keyParts) == 2 {
		envVar.KeyPath = keyParts[1]
		if strings.Index(envVar.KeyPath, "_") == 0 {
			envVar.KeyPath = envVar.KeyPath[1:]
			envVar.ShouldConf = false
		}
		if strings.Contains(keyParts[0], "U") {
			envVar.IsUpstream = true
		}
	}

	return envVar
}

type EnvVarMeta struct {
}

type EnvVar struct {
	KeyPath    string
	Value      string
	Mask       string
	ShouldConf bool
	IsUpstream bool
}

func (envVar *EnvVar) Key(prefix string) string {
	if envVar.IsUpstream {
		prefix += "U"
	}
	if !envVar.ShouldConf {
		prefix += "_"
	}
	return prefix + "__" + envVar.KeyPath
}
