package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const CACHE_DURATION = 5 * time.Minute

type TempoInstance struct {
	Kind      KindType `json:"kind"`
	Namespace string   `json:"tempoNamespace"`
	Name      string   `json:"tempoName"`
	Tenants   []string `json:"tenants,omitempty"`
	Status    string   `json:"status"`
	hostname  string   // if useRoute is enabled: host of the route, otherwise the DNS name of the k8s service
}

type KindType string

const (
	KindTempoStack      KindType = "TempoStack"
	KindTempoMonolithic KindType = "TempoMonolithic"
)

type TempoDiscovery struct {
	k8sClient *dynamic.DynamicClient
	useRoute  bool

	cachedInstances []TempoInstance
	cacheExpiry     time.Time
}

// The k8sClient must have permission to list TempoStack and TempoMonolithic resources cluster-wide.
func New(k8sClient *dynamic.DynamicClient, useRoute bool) *TempoDiscovery {
	return &TempoDiscovery{
		k8sClient: k8sClient,
		useRoute:  useRoute,
	}
}

func (d *TempoDiscovery) ListInstances(ctx context.Context) ([]TempoInstance, error) {
	if time.Now().Before(d.cacheExpiry) {
		return d.cachedInstances, nil
	}

	slog.Debug("fetching TempoStack and TempoMonolithic instances from cluster")
	tempos := []TempoInstance{}

	tempoStacks, err := d.listTempoStacks(ctx)
	if err != nil {
		return nil, err
	}
	tempos = append(tempos, tempoStacks...)

	tempoMonolithics, err := d.listTempoMonolithics(ctx)
	if err != nil {
		return nil, err
	}
	tempos = append(tempos, tempoMonolithics...)

	d.cachedInstances = tempos
	d.cacheExpiry = time.Now().Add(CACHE_DURATION)

	return tempos, nil
}

func (d *TempoDiscovery) listTempoStacks(ctx context.Context) ([]TempoInstance, error) {
	list, err := d.k8sClient.Resource(tempoStackGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list TempoStacks: %w", err)
	}

	var instances []TempoInstance
	for _, item := range list.Items {
		var tempo TempoStack
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &tempo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TempoStack: %w", err)
		}

		if tempo.Spec.Tenants == nil || tempo.Spec.Tenants.Mode != ModeOpenShift {
			continue
		}

		tenants := make([]string, 0, len(tempo.Spec.Tenants.Authentication))
		for _, auth := range tempo.Spec.Tenants.Authentication {
			tenants = append(tenants, auth.TenantName)
		}

		status := getStatusFromConditions(tempo.Status.Conditions)
		hostname, err := d.getHostname(ctx, tempo.Namespace, tempo.Name)
		if err != nil {
			return nil, err
		}

		instances = append(instances, TempoInstance{
			Kind:      KindTempoStack,
			Namespace: tempo.Namespace,
			Name:      tempo.Name,
			Tenants:   tenants,
			Status:    status,
			hostname:  hostname,
		})
	}

	return instances, nil
}

func (d *TempoDiscovery) listTempoMonolithics(ctx context.Context) ([]TempoInstance, error) {
	list, err := d.k8sClient.Resource(tempoMonolithicGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list TempoMonolithics: %w", err)
	}

	var instances []TempoInstance
	for _, item := range list.Items {
		var tempo TempoMonolithic
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &tempo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TempoMonolithic: %w", err)
		}

		if tempo.Spec.Multitenancy == nil || !tempo.Spec.Multitenancy.Enabled || tempo.Spec.Multitenancy.Mode != ModeOpenShift {
			continue
		}

		tenants := make([]string, 0, len(tempo.Spec.Multitenancy.Authentication))
		for _, auth := range tempo.Spec.Multitenancy.Authentication {
			tenants = append(tenants, auth.TenantName)
		}

		status := getStatusFromConditions(tempo.Status.Conditions)
		hostname, err := d.getHostname(ctx, tempo.Namespace, tempo.Name)
		if err != nil {
			return nil, err
		}

		instances = append(instances, TempoInstance{
			Kind:      KindTempoMonolithic,
			Namespace: tempo.Namespace,
			Name:      tempo.Name,
			Tenants:   tenants,
			Status:    status,
			hostname:  hostname,
		})
	}

	return instances, nil
}

func getStatusFromConditions(conditions []metav1.Condition) string {
	for _, cond := range conditions {
		if cond.Status == metav1.ConditionTrue {
			return cond.Type
		}
	}
	return ""
}

func (d *TempoDiscovery) getHostname(ctx context.Context, namespace, name string) (string, error) {
	serviceName := DNSName(fmt.Sprintf("tempo-%s-gateway", name))
	if !d.useRoute {
		return fmt.Sprintf("%s.%s.svc", serviceName, namespace), nil
	}

	// fetch the route and extract the host field from the spec
	routeName := serviceName
	unstructured, err := d.k8sClient.Resource(routeGVR).Namespace(namespace).Get(ctx, routeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get route %s/%s: %w", namespace, routeName, err)
	}

	var route Route
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &route)
	if err != nil {
		return "", fmt.Errorf("failed to parse route %s/%s: %w", namespace, routeName, err)
	}

	return route.Spec.Host, nil
}

func (t *TempoInstance) GetURL(tenant string) string {
	return fmt.Sprintf("https://%s/api/traces/v1/%s/tempo", t.hostname, url.PathEscape(tenant))
}
