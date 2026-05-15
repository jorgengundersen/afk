package afk

import (
	"fmt"
	"strings"
)

func EnvForNonItemInvocation(parent []string, iteration int) []string {
	env := make([]string, 0, len(parent)+1)
	for _, entry := range parent {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if key == "AFK_INDEX" || key == "AFK_ITEM" || key == "AFK_ITEM_INDEX" || key == "AFK_ITEM_COUNT" {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, fmt.Sprintf("AFK_INDEX=%d", iteration))
	return env
}
