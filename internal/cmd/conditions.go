package cmd

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
)

func (o *waitxOptions) lookupConditions(ctx context.Context, resourceArg string) []string {
	infos, err := o.resourceInfos(ctx, resourceArg)
	if err != nil {
		return nil
	}
	for _, info := range infos {
		if conditions := builtinConditionsFor(info); len(conditions) != 0 {
			return conditions
		}
		u, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		// Fall back to the first object that already has conditions.
		if conditions := extractConditionTypes(u); len(conditions) != 0 {
			return conditions
		}
	}
	return nil
}

func completionResourceArg(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	resource := args[0]
	if len(args) == 1 || strings.Contains(resource, "/") {
		return resource, true
	}
	return resource + "/" + args[1], true
}

func extractConditionTypes(u *unstructured.Unstructured) []string {
	items, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}

	conditions := make([]string, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if condition, _ := entry["type"].(string); condition != "" {
			conditions = append(conditions, condition)
		}
	}
	return conditions
}

func builtinConditionsFor(info *resource.Info) []string {
	if info.Mapping == nil {
		return nil
	}
	return builtinConditions[info.Mapping.GroupVersionKind.Kind]
}

func (o *waitxOptions) resourceInfos(ctx context.Context, resourceArg string) ([]*resource.Info, error) {
	if o.resourceInfosFunc != nil {
		return o.resourceInfosFunc(ctx, resourceArg)
	}
	factory := o.factoryFunc()
	namespace, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	return factory.NewBuilder().
		Unstructured().
		DefaultNamespace().
		NamespaceParam(namespace).
		ResourceTypeOrNameArgs(true, resourceArg).
		Latest().
		Flatten().
		Do().
		Infos()
}
