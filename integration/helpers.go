package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	aoapis "github.com/openshift/addon-operator/apis"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"

	referenceaddonapis "github.com/openshift/reference-addon/apis"
)

const (
	RelativeConfigDeployPath = "../config/deploy"
	RelativeConfigAddonPath  = "../config/addon"
)

var (
	// Client pointing to the e2e test cluster.
	Client client.Client
	Config *rest.Config
	Scheme = runtime.NewScheme()

	// Typed K8s Clients
	CoreV1Client corev1client.CoreV1Interface
)

func init() {
	schemesToBeAdded := runtime.SchemeBuilder{
		clientgoscheme.AddToScheme,
		aoapis.AddToScheme,
		apiextensionsv1.AddToScheme,
		referenceaddonapis.AddToScheme,
	}

	if err := schemesToBeAdded.AddToScheme(Scheme); err != nil {
		panic(fmt.Errorf("could not load schemes: %w", err))
	}

	Config = ctrl.GetConfigOrDie()

	var err error
	Client, err = client.New(Config, client.Options{
		Scheme: Scheme,
	})
	if err != nil {
		panic(err)
	}

	// Typed Kubernetes Clients
	CoreV1Client = corev1client.NewForConfigOrDie(Config)
}

type fileInfoMap struct {
	absPath  string
	fileInfo []os.FileInfo
}

type fileInfosByName []os.FileInfo

func (x fileInfosByName) Len() int { return len(x) }

func (x fileInfosByName) Less(i, j int) bool {
	iName := path.Base(x[i].Name())
	jName := path.Base(x[j].Name())
	return iName < jName
}
func (x fileInfosByName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func getFileInfoFromPath(paths []string) ([]fileInfoMap, error) {
	fileInfo := []fileInfoMap{}
	for _, path := range paths {
		config, err := os.Open(path)
		if err != nil {
			return fileInfo, err
		}

		files, err := config.Readdir(-1)
		if err != nil {
			return fileInfo, err
		}
		sort.Sort(fileInfosByName(files))
		fileInfo = append(fileInfo, fileInfoMap{
			absPath:  path,
			fileInfo: files,
		})
	}
	return fileInfo, nil
}

func LoadObjectsFromDirectory(t *testing.T, directoryPath string) []unstructured.Unstructured {
	configDeployPath, err := filepath.Abs(directoryPath)
	if err != nil {
		return []unstructured.Unstructured{}
	}
	var objects []unstructured.Unstructured
	paths := []string{configDeployPath}
	fileInfoMap, err := getFileInfoFromPath(paths)
	require.NoError(t, err)

	for _, m := range fileInfoMap {
		for _, f := range m.fileInfo {
			if f.IsDir() {
				continue
			}
			if path.Ext(f.Name()) != ".yaml" {
				continue
			}

			fileYaml, err := ioutil.ReadFile(path.Join(
				m.absPath, f.Name()))
			require.NoError(t, err)

			// Trim empty starting and ending objects
			fileYaml = bytes.Trim(fileYaml, "---\n")

			// Split for every included yaml document.
			for _, yamlDocument := range bytes.Split(fileYaml, []byte("---\n")) {
				obj := unstructured.Unstructured{}
				require.NoError(t, yaml.Unmarshal(yamlDocument, &obj))

				objects = append(objects, obj)
			}
		}
	}

	return objects

}

// WaitToBeGone blocks until the given object is gone from the kubernetes API server.
func WaitToBeGone(t *testing.T, timeout time.Duration, object client.Object) error {
	gvk, err := apiutil.GVKForObject(object, Scheme)
	if err != nil {
		return err
	}

	key := client.ObjectKeyFromObject(object)
	t.Logf("waiting %s for %s %s to be gone...",
		timeout, gvk, key)

	ctx := context.Background()
	defaultWaitPollInterval := 1 * time.Second
	return wait.PollImmediate(defaultWaitPollInterval, timeout, func() (done bool, err error) {
		err = Client.Get(ctx, key, object)

		if errors.IsNotFound(err) {
			return true, nil
		}

		if err != nil {
			t.Logf("error waiting for %s %s to be gone: %v",
				object.GetObjectKind().GroupVersionKind().Kind, key, err)
		}
		return false, nil
	})
}

func WaitForObject(
	t *testing.T, timeout time.Duration,
	object client.Object, reason string,
	checkFn func(obj client.Object) (done bool, err error),
) error {
	gvk, err := apiutil.GVKForObject(object, Scheme)
	if err != nil {
		return err
	}

	key := client.ObjectKeyFromObject(object)
	t.Logf("waiting %s on %s %s %s...",
		timeout, gvk, key, reason)

	ctx := context.Background()
	return wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
		err = Client.Get(ctx, client.ObjectKeyFromObject(object), object)
		if err != nil {
			//nolint:nilerr // retry on transient errors
			return false, nil
		}

		return checkFn(object)
	})
}
