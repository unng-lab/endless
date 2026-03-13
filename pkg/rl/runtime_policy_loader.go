package rl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type runtimeArtifactKind string

const (
	runtimeArtifactKindLinearQStub runtimeArtifactKind = "linear_q_stub"
	runtimeArtifactKindGoMLXCritic runtimeArtifactKind = "gomlx_critic"
)

// LoadRuntimePolicyFromPath auto-detects which serialized model format was passed through the
// launcher flag and instantiates the matching runtime policy implementation.
func LoadRuntimePolicyFromPath(path string) (Policy, string, error) {
	kind, err := detectRuntimeArtifactKind(path)
	if err != nil {
		return nil, "", err
	}

	switch kind {
	case runtimeArtifactKindLinearQStub:
		policy, err := LoadLinearQStubRuntimePolicy(path)
		if err != nil {
			return nil, "", err
		}
		return policy, string(runtimeArtifactKindLinearQStub), nil
	case runtimeArtifactKindGoMLXCritic:
		policy, err := LoadGoMLXCriticRuntimePolicy(path)
		if err != nil {
			return nil, "", err
		}
		return policy, string(runtimeArtifactKindGoMLXCritic), nil
	default:
		return nil, "", fmt.Errorf("unsupported runtime artifact kind %q", kind)
	}
}

// detectRuntimeArtifactKind distinguishes the supported runtime model inputs so the launcher may
// route one user-supplied path to the correct loader without asking for extra flags.
func detectRuntimeArtifactKind(path string) (runtimeArtifactKind, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("runtime artifact path is empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat runtime artifact path %q: %w", path, err)
	}
	if info.IsDir() {
		manifestPath := filepath.Join(path, goMLXCriticManifestFileName)
		if _, err := os.Stat(manifestPath); err == nil {
			return runtimeArtifactKindGoMLXCritic, nil
		}
		return "", fmt.Errorf("runtime artifact directory %q does not contain %q", path, goMLXCriticManifestFileName)
	}

	lowerPath := strings.ToLower(path)
	switch filepath.Ext(lowerPath) {
	case ".bin":
		return runtimeArtifactKindGoMLXCritic, nil
	case ".json":
		root, err := loadRuntimeArtifactJSONRoot(path)
		if err != nil {
			return "", err
		}
		switch {
		case hasJSONKeys(root, "trained_at", "obs_dim", "action_dim", "input_dim", "checkpoint_dir"):
			return runtimeArtifactKindGoMLXCritic, nil
		case hasJSONKeys(root, "Variables", "BinFormat"):
			return runtimeArtifactKindGoMLXCritic, nil
		case hasJSONKeys(root, "saved_at", "normalization_spec", "model"):
			return runtimeArtifactKindLinearQStub, nil
		default:
			// Keep the historical default for unknown JSON artifacts so existing linear stub paths
			// still take the original validation path.
			return runtimeArtifactKindLinearQStub, nil
		}
	default:
		return "", fmt.Errorf("runtime artifact path %q must be a JSON file, GoMLX checkpoint binary, or GoMLX checkpoint directory", path)
	}
}

func loadRuntimeArtifactJSONRoot(path string) (map[string]json.RawMessage, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read runtime artifact %q: %w", path, err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("decode runtime artifact %q as JSON: %w", path, err)
	}
	return root, nil
}
