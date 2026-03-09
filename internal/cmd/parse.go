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
		case strings.HasPrefix(arg, "--for=condition="):
			req.mode = completionModeForValue
			req.conditionContext = true
			req.forValue = strings.TrimPrefix(arg, "--for=condition=")
			return req
		case strings.HasPrefix(arg, "--for="):
			req.mode = completionModeForValue
			req.forValue = strings.TrimPrefix(arg, "--for=")
			req.conditionContext = strings.HasPrefix(req.forValue, "condition=")
			if req.conditionContext {
				req.forValue = strings.TrimPrefix(req.forValue, "condition=")
			}
			return req
		case arg == "--for":
			if i+1 >= len(args) {
				req.mode = completionModeForFlag
				return req
			}
			req.mode = completionModeForValue
			value := args[i+1]
			if strings.HasPrefix(value, "condition=") {
				req.conditionContext = true
				req.valuePrefix = "condition="
				req.forValue = strings.TrimPrefix(value, "condition=")
			} else {
				req.forValue = value
			}
			return req
		case strings.HasPrefix(arg, "-"):
			req.mode = completionModeFlagPartial
			req.toComplete = arg
		default:
			req.resourceArgs = append(req.resourceArgs, arg)
			req.toComplete = arg
		}
	}
	return req
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
