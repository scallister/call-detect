//go:build linux

package live

import (
	"encoding/json"
	"fmt"
	"strings"
)

func pipewireDump() ([]pwNode, error) {
	raw, err := runCmd("pw-dump")
	if err != nil {
		return nil, fmt.Errorf("pw-dump: %w", err)
	}
	return parsePWDump(raw)
}

func pipewireClassNames(class string) ([]string, error) {
	nodes, err := pipewireDump()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, n := range nodes {
		if !strings.EqualFold(n.mediaClass, class) {
			continue
		}
		if name := firstProp(n.props, "application.process.binary", "application.name", "node.name"); name != "" {
			names = append(names, name)
		}
	}
	return uniqueNames(names), nil
}

type pwNode struct {
	mediaClass string
	props      map[string]string
}

func parsePWDump(raw []byte) ([]pwNode, error) {
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("pw-dump json: %w", err)
	}
	var out []pwNode
	for _, row := range rows {
		typ, _ := row["type"].(string)
		if typ != "" && !strings.Contains(typ, "Node") {
			continue
		}
		props := map[string]string{}
		if info, ok := row["info"].(map[string]any); ok {
			for k, v := range stringMap(info["props"]) {
				props[k] = v
			}
			for k, v := range stringMap(info["properties"]) {
				props[k] = v
			}
		}
		for k, v := range stringMap(row["props"]) {
			props[k] = v
		}
		if len(props) == 0 {
			continue
		}
		class := firstProp(props, "media.class")
		if class == "" {
			continue
		}
		out = append(out, pwNode{mediaClass: class, props: props})
	}
	return out, nil
}
