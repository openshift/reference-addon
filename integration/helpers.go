package integration

import (
	"fmt"

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

func generateRBAC(group, namespace string) []client.Object {
	return []client.Object{
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      group,
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
				Name: group,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      group,
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"events",
					},
					Verbs: []string{
						"create",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"secrets",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"configmaps",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"coordination.k8s.io",
					},
					Resources: []string{
						"leases",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"create",
						"update",
						"patch",
						"delete",
					},
				},
				{
					APIGroups: []string{
						"reference.addons.managed.openshift.io",
					},
					Resources: []string{
						"referenceaddons",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"create",
						"update",
						"patch",
						"delete",
					},
				},
				{
					APIGroups: []string{
						"operators.coreos.com",
					},
					Resources: []string{
						"clusterserviceversions",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"delete",
					},
				},
				{
					APIGroups: []string{
						"networking.k8s.io",
					},
					Resources: []string{
						"networkpolicies",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"create",
						"update",
						"delete",
					},
				},
			},
		},
	}
}
