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
			req.mode = completionModeForValue
			req.conditionContext, req.forValue, req.valuePrefix = parseForValue(strings.TrimPrefix(arg, "--for="), false)
			return req
		case arg == "--for":
			if i+1 >= len(args) {
				req.mode = completionModeForFlag
				return req
			}
			req.mode = completionModeForValue
			req.conditionContext, req.forValue, req.valuePrefix = parseForValue(args[i+1], true)
			return req
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

func completionResourceArg(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	if len(args) == 1 {
		if strings.Contains(args[0], "/") {
			return args[0], true
		}
		return "", false
	}
	if strings.Contains(args[0], "/") {
		return args[0], true
	}
	return args[0] + "/" + args[1], true
}
