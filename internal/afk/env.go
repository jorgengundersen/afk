package afk

import (
	"fmt"
	"strings"
)

func EnvForNonItemInvocation(parent []string, iteration int) []string {
	env := baseEnvWithoutAFKContext(parent)
	env = append(env, fmt.Sprintf("AFK_INDEX=%d", iteration))
	return env
}

func EnvForItemInvocation(parent []string, iteration int, item string, itemIndex int, itemCount int) []string {
	env := baseEnvWithoutAFKContext(parent)
	env = append(env,
		fmt.Sprintf("AFK_INDEX=%d", iteration),
		fmt.Sprintf("AFK_ITEM=%s", item),
		fmt.Sprintf("AFK_ITEM_INDEX=%d", itemIndex),
		fmt.Sprintf("AFK_ITEM_COUNT=%d", itemCount),
	)
	return env
}

func baseEnvWithoutAFKContext(parent []string) []string {
	env := make([]string, 0, len(parent)+4)
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
	return env
}
