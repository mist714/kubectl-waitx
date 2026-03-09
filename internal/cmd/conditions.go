package cmd

import (
	"context"
	"errors"
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
)

func (o *waitxOptions) lookupConditions(ctx context.Context, resourceArg string) ([]string, error) {
	if o.conditionLookupFunc != nil {
		return o.conditionLookupFunc(ctx, resourceArg)
	}

	infos, err := o.resourceInfos(ctx, resourceArg)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, errors.New("resource not found")
	}

	seen := map[string]struct{}{}
	for _, info := range infos {
		object, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		items, found, err := unstructured.NestedSlice(object.Object, "status", "conditions")
		if err != nil || !found {
			continue
		}
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value, ok := entry["type"].(string)
			if ok && value != "" {
				seen[value] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return nil, errors.New("no conditions found")
	}

	conditions := make([]string, 0, len(seen))
	for condition := range seen {
		conditions = append(conditions, condition)
	}
	slices.Sort(conditions)
	return conditions, nil
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
