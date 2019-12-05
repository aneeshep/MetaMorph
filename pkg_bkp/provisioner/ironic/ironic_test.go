package ironic

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"

	metalkubev1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metalkube/v1alpha1"
	"github.com/metal3-io/baremetal-operator/pkg/bmc"
)

func init() {
	logf.SetLogger(logf.ZapLogger(true))
}

func TestChecksumIsURLNo(t *testing.T) {
	isURL, err := checksumIsURL("checksum-goes-here")
	if isURL {
		t.Fail()
	}
	if err != nil {
		t.Fail()
	}
}

func TestChecksumIsURLYes(t *testing.T) {
	isURL, err := checksumIsURL("http://checksum-goes-here")
	if !isURL {
		t.Fail()
	}
	if err != nil {
		t.Fail()
	}
}

func TestGetUpdateOptsForNodeVirtual(t *testing.T) {
	host := &metalkubev1alpha1.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myhost",
			Namespace: "myns",
		},
		Spec: metalkubev1alpha1.BareMetalHostSpec{
			Image: &metalkubev1alpha1.Image{
				URL: "not-empty",
			},
			Online: true,
		},
		Status: metalkubev1alpha1.BareMetalHostStatus{
			HardwareProfile: "libvirt",
			Provisioning: metalkubev1alpha1.ProvisionStatus{
				ID: "provisioning-id",
			},
		},
	}

	eventPublisher := func(reason, message string) {}

	prov, err := newProvisioner(host, bmc.Credentials{}, eventPublisher)
	ironicNode := &nodes.Node{}

	patches, err := prov.getUpdateOptsForNode(ironicNode, "checksum")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("patches: %v", patches)

	var update nodes.UpdateOperation

	expected := []struct {
		Path  string      // the node property path
		Key   string      // if value is a map, the key we care about
		Value interface{} // the value being passed to ironic (or value associated with the key)
	}{
		{
			Path:  "/instance_info/image_source",
			Value: "not-empty",
		},
		{
			Path:  "/instance_info/image_checksum",
			Value: "checksum",
		},
		{
			Path:  "/instance_uuid",
			Value: "provisioning-id",
		},
		{
			Path:  "/instance_info/root_gb",
			Value: 10,
		},
		{
			Path:  "/instance_info/root_device",
			Value: "/dev/vda",
			Key:   "name",
		},
		{
			Path:  "/properties/cpu_arch",
			Value: "x86_64",
		},
		{
			Path:  "/properties/local_gb",
			Value: 50,
		},
	}

	for i, e := range expected {
		update = patches[i].(nodes.UpdateOperation)
		if e.Key != "" {
			m := update.Value.(map[string]string)
			if m[e.Key] != e.Value {
				t.Errorf("expected %s=%q got %s=%q", e.Path, e.Value, update.Path, update.Value)
			}
		} else {
			if update.Value != e.Value {
				t.Errorf("expected %s=%q got %s=%q", e.Path, e.Value, update.Path, update.Value)
			}
		}
	}
}

func TestGetUpdateOptsForNodeDell(t *testing.T) {
	host := &metalkubev1alpha1.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myhost",
			Namespace: "myns",
		},
		Spec: metalkubev1alpha1.BareMetalHostSpec{
			Image: &metalkubev1alpha1.Image{
				URL: "not-empty",
			},
			Online: true,
		},
		Status: metalkubev1alpha1.BareMetalHostStatus{
			HardwareProfile: "dell",
			Provisioning: metalkubev1alpha1.ProvisionStatus{
				ID: "provisioning-id",
			},
		},
	}

	eventPublisher := func(reason, message string) {}

	prov, err := newProvisioner(host, bmc.Credentials{}, eventPublisher)
	ironicNode := &nodes.Node{}

	patches, err := prov.getUpdateOptsForNode(ironicNode, "checksum")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("patches: %v", patches)

	var update nodes.UpdateOperation

	expected := []struct {
		Path  string      // the node property path
		Key   string      // if value is a map, the key we care about
		Value interface{} // the value being passed to ironic (or value associated with the key)
	}{
		{
			Path:  "/instance_info/image_source",
			Value: "not-empty",
		},
		{
			Path:  "/instance_info/image_checksum",
			Value: "checksum",
		},
		{
			Path:  "/instance_uuid",
			Value: "provisioning-id",
		},
		{
			Path:  "/instance_info/root_gb",
			Value: 10,
		},
		{
			Path:  "/instance_info/root_device",
			Value: "0:0:0:0",
			Key:   "hctl",
		},
		{
			Path:  "/properties/cpu_arch",
			Value: "x86_64",
		},
		{
			Path:  "/properties/local_gb",
			Value: 50,
		},
	}

	for i, e := range expected {
		update = patches[i].(nodes.UpdateOperation)
		if e.Key != "" {
			m := update.Value.(map[string]string)
			if m[e.Key] != e.Value {
				t.Errorf("expected %s=%q got %s=%q", e.Path, e.Value, update.Path, update.Value)
			}
		} else {
			if update.Value != e.Value {
				t.Errorf("expected %s=%q got %s=%q", e.Path, e.Value, update.Path, update.Value)
			}
		}
	}
}
