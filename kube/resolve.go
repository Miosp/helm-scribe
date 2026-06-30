package kube

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var versionPattern = regexp.MustCompile(`^v[0-9]+((alpha|beta)[0-9]+)?$`)

var (
	indexOnce sync.Once
	gvkIndex  map[string][]string // "group/version/Kind" -> full definition keys
)

func buildIndex() {
	gvkIndex = make(map[string][]string)
	for key := range Definitions() {
		segs := strings.Split(key, ".")
		if len(segs) < 3 {
			continue
		}
		version := segs[len(segs)-2]
		if !versionPattern.MatchString(version) {
			continue
		}
		group := segs[len(segs)-3]
		kind := segs[len(segs)-1]
		canonical := group + "/" + version + "/" + kind
		gvkIndex[canonical] = append(gvkIndex[canonical], key)
	}
}

// Resolve maps a short GVK ("core/v1/Container", "networking.k8s.io/v1/Ingress")
// to its fully-qualified definition key. It returns an error when the type is
// unknown or matches more than one definition.
func Resolve(gvk string) (string, error) {
	indexOnce.Do(buildIndex)

	parts := strings.Split(gvk, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", fmt.Errorf("invalid Kubernetes type %q; expected group/version/Kind", gvk)
	}
	group := strings.SplitN(parts[0], ".", 2)[0]
	canonical := group + "/" + parts[1] + "/" + parts[2]

	matches := gvkIndex[canonical]
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("unknown Kubernetes type %q", gvk)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous Kubernetes type %q matches %s", gvk, strings.Join(matches, ", "))
	}
}
