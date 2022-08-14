package cluster

// Hooks define a list of hooks such as hooks["apply"]["before"] = ["ls -al", "rm foo.txt"]
type Hooks map[string]map[string][]string

// ForActionAndStage return hooks for given action and stage
func (h Hooks) ForActionAndStage(action, stage string) []string {
	if len(h[action]) > 0 {
		return h[action][stage]
	}
	return nil
}
