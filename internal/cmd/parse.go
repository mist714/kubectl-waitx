package cmd

import "strings"

func parseCompletionRequest(args []string) completionRequest {
	req := completionRequest{mode: completionModeResource}
	if len(args) == 0 {
		return req
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--for="):
			req.applyForValue(strings.TrimPrefix(arg, "--for="), false, arg)
		case arg == "--for":
			if i+1 >= len(args) {
				req.mode = completionModeForFlag
				return req
			}
			req.applyForValue(args[i+1], true, args[i+1])
			i++
		case forArgMode(arg) == completionModeFlagPartial:
			req.mode = completionModeFlagPartial
			req.toComplete = arg
		default:
			req.resourceArgs = append(req.resourceArgs, arg)
			req.toComplete = arg
		}
	}
	return req
}

func (req *completionRequest) applyForValue(value string, separate bool, toComplete string) {
	req.conditionContext, req.forValue, req.valuePrefix = parseForValue(value, separate)
	req.mode = completionModeForValue
	req.toComplete = toComplete
}

func forArgMode(arg string) completionMode {
	if strings.HasPrefix(arg, "-") {
		return completionModeFlagPartial
	}
	return completionModeResource
}

func parseForValue(value string, separate bool) (conditionContext bool, forValue, valuePrefix string) {
	if strings.HasPrefix(value, "condition=") {
		if separate {
			return true, strings.TrimPrefix(value, "condition="), "condition="
		}
		return true, strings.TrimPrefix(value, "condition="), ""
	}
	return false, value, ""
}
