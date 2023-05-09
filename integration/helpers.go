package integration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	//"github.com/openshift/reference-addon/internal/controllers/addoninstance"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func nameGenerator(pfx string) func() string {
	i := 0

	return func() string {
		name := fmt.Sprintf("%s-%d", pfx, i)

		i++

		return name
	}
}

func getRBAC(namespace, group string) ([]client.Object, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, err
	}

	role, err := internaltesting.LoadUnstructuredFromFile(filepath.Join(root, "config", "deploy", "role.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading role: %w", err)
	}

	role.SetNamespace(namespace)

	return []client.Object{
		role,
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      role.GetName(),
				Namespace: namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: group,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: role.GetName(),
			},
		},
	}, nil
}

func projectRoot() (string, error) {
	var buf bytes.Buffer

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout = &buf
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("determining top level directory from git: %w", errSetup)
	}

	return strings.TrimSpace(buf.String()), nil
}

var errSetup = errors.New("test setup failed")

func remove(path string) error {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil
	}

	return os.Remove(path)
}

func addonParameterSecret(name, ns string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func addonNetworkPolicy(name, ns string) netv1.NetworkPolicy {
	return netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	}
}

func addonInstanceObject(name, ns string) addonsv1alpha1.AddonInstance {
	return addonsv1alpha1.AddonInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}
