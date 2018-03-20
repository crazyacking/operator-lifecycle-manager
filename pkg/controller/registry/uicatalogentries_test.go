package registry

import (
	"testing"

	csvv1alpha1 "github.com/coreos-inc/alm/pkg/api/apis/clusterserviceversion/v1alpha1"
	"github.com/coreos-inc/alm/pkg/api/apis/uicatalogentry/v1alpha1"

	catsrcv1alpha1 "github.com/coreos-inc/alm/pkg/api/apis/catalogsource/v1alpha1"
	"github.com/coreos-inc/alm/pkg/api/client/clientset/versioned/fake"
	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/client-go/testing"
)

func RequireActions(t *testing.T, expected, actual []core.Action) {
	require.EqualValues(t, len(expected), len(actual), "Expected\n\t%#v\ngot\n\t%#v", expected, actual)
	for i, a := range actual {
		e := expected[i]
		if _, ok := e.(core.CreateAction); ok {
			require.True(t, equality.Semantic.DeepEqual(e.(core.CreateAction).GetObject(), a.(core.CreateAction).GetObject()), "Expected\n\t%#v\ngot\n\t%#v", e.(core.CreateAction).GetObject(), a.(core.CreateAction).GetObject())
		}
		require.True(t, equality.Semantic.DeepEqual(e, a), "Expected\n\t%#v\ngot\n\t%#v", e, a)
	}
}

func RequireSemanticEqual(t *testing.T, expected, actual interface{}) {
	require.True(t, equality.Semantic.DeepEqual(expected, actual), "Expected\n\t%#v\ngot\n\t%#v", expected, actual)
}

func makeCRDs(names ...string) []*v1beta1.CustomResourceDefinition {
	crds := []*v1beta1.CustomResourceDefinition{}
	for _, name := range names {
		crds = append(crds, &v1beta1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "CustomResourceDefinition",
			},
			Spec: v1beta1.CustomResourceDefinitionSpec{
				Group:   name + "-group",
				Version: name + "version",
			},
		})
	}
	return crds
}

func makeCSV(name string, version string, ownedCRDs, requiredCRDs []*v1beta1.CustomResourceDefinition) *csvv1alpha1.ClusterServiceVersion {
	csv := &csvv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:     name,
			SelfLink: "/link/" + name,
		},
		Spec: csvv1alpha1.ClusterServiceVersionSpec{
			Version:     *semver.New(version),
			DisplayName: name,
			CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
				Owned:    []csvv1alpha1.CRDDescription{},
				Required: []csvv1alpha1.CRDDescription{},
			},
		},
	}

	for _, owned := range ownedCRDs {
		csv.Spec.CustomResourceDefinitions.Owned = append(csv.Spec.CustomResourceDefinitions.Owned, csvv1alpha1.CRDDescription{
			Name:    owned.Name,
			Version: owned.APIVersion,
			Kind:    owned.Kind,
		})
	}
	for _, required := range requiredCRDs {
		csv.Spec.CustomResourceDefinitions.Required = append(csv.Spec.CustomResourceDefinitions.Required, csvv1alpha1.CRDDescription{
			Name:    required.Name,
			Version: required.APIVersion,
			Kind:    required.Kind,
		})
	}

	return csv
}

func uiCatalogEntry(csv *csvv1alpha1.ClusterServiceVersion, manifest v1alpha1.PackageManifest, ownerRefs []metav1.OwnerReference) *v1alpha1.UICatalogEntry {
	return &v1alpha1.UICatalogEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.UICatalogEntryKind,
			APIVersion: v1alpha1.UICatalogEntryCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            manifest.PackageName,
			Namespace:       "alm-coreos-tests",
			Labels:          map[string]string{"tectonic-visibility": "ocs"},
			OwnerReferences: ownerRefs,
		},
		Spec: &v1alpha1.UICatalogEntrySpec{
			Manifest: manifest,
			CSVSpec:  csv.Spec,
		},
	}
}

func TestCustomCatalogStore(t *testing.T) {
	source := catsrcv1alpha1.CatalogSource{}
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(&source, source.GroupVersionKind()),
	}

	testPackageName := "MockServiceName"
	testCSVName := "MockServiceName-v1"
	testCSVVersion := "0.2.4+alpha"

	manifest := v1alpha1.PackageManifest{
		PackageName: testPackageName,
		Channels: []v1alpha1.PackageChannel{
			{
				Name:           "alpha",
				CurrentCSVName: testCSVName,
			},
		},
	}
	csv := csvv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       csvv1alpha1.ClusterServiceVersionCRDName,
			APIVersion: csvv1alpha1.GroupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        testCSVName,
			Namespace:   "alm-coreos-tests",
			Annotations: map[string]string{"tectonic-visibility": "tectonic-feature"},
		},
		Spec: csvv1alpha1.ClusterServiceVersionSpec{
			Version: *semver.New(testCSVVersion),
			CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
				Owned:    []csvv1alpha1.CRDDescription{},
				Required: []csvv1alpha1.CRDDescription{},
			},
		},
	}
	expectedEntry := &v1alpha1.UICatalogEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.UICatalogEntryKind,
			APIVersion: v1alpha1.UICatalogEntryCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            testPackageName,
			Namespace:       "alm-coreos-tests",
			Labels:          map[string]string{"tectonic-visibility": "tectonic-feature"},
			OwnerReferences: ownerRefs,
		},
		Spec: &v1alpha1.UICatalogEntrySpec{
			Manifest: v1alpha1.PackageManifest{
				PackageName: testPackageName,
				Channels: []v1alpha1.PackageChannel{
					{
						Name:           "alpha",
						CurrentCSVName: testCSVName,
					},
				},
			},
			CSVSpec: csvv1alpha1.ClusterServiceVersionSpec{
				Version: *semver.New(testCSVVersion),
				CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
					Owned:    []csvv1alpha1.CRDDescription{},
					Required: []csvv1alpha1.CRDDescription{},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset()
	store := CustomResourceCatalogStore{Client: fakeClient, Namespace: "alm-coreos-tests"}

	actualEntry, err := store.Store(manifest, &csv, ownerRefs)
	require.NoError(t, err)

	expectedActions := []core.Action{
		core.NewGetAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry.GetName(),
		),
		core.NewCreateAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry,
		),
		core.NewGetAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry.GetName(),
		),
	}

	returnEntry, err := fakeClient.UicatalogentryV1alpha1().UICatalogEntries(expectedEntry.GetNamespace()).Get(expectedEntry.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
	RequireSemanticEqual(t, returnEntry, actualEntry)
	RequireActions(t, expectedActions, fakeClient.Actions())
}

func TestCustomCatalogStoreDefaultVisibility(t *testing.T) {
	source := catsrcv1alpha1.CatalogSource{}
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(&source, source.GroupVersionKind()),
	}

	testPackageName := "MockServiceName"
	testCSVName := "MockServiceName-v1"
	testCSVVersion := "0.2.4+alpha"

	manifest := v1alpha1.PackageManifest{
		PackageName: testPackageName,
		Channels: []v1alpha1.PackageChannel{
			{
				Name:           "alpha",
				CurrentCSVName: testCSVName,
			},
		},
	}
	csv := csvv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       csvv1alpha1.ClusterServiceVersionCRDName,
			APIVersion: csvv1alpha1.GroupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCSVName,
			Namespace: "alm-coreos-tests",
		},
		Spec: csvv1alpha1.ClusterServiceVersionSpec{
			Version: *semver.New(testCSVVersion),
			CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
				Owned:    []csvv1alpha1.CRDDescription{},
				Required: []csvv1alpha1.CRDDescription{},
			},
		},
	}
	expectedEntry := &v1alpha1.UICatalogEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.UICatalogEntryKind,
			APIVersion: v1alpha1.UICatalogEntryCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            testPackageName,
			Namespace:       "alm-coreos-tests",
			Labels:          map[string]string{"tectonic-visibility": "ocs"},
			OwnerReferences: ownerRefs,
		},
		Spec: &v1alpha1.UICatalogEntrySpec{
			Manifest: v1alpha1.PackageManifest{
				PackageName: testPackageName,
				Channels: []v1alpha1.PackageChannel{
					{
						Name:           "alpha",
						CurrentCSVName: testCSVName,
					},
				},
			},
			CSVSpec: csvv1alpha1.ClusterServiceVersionSpec{
				Version: *semver.New(testCSVVersion),
				CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
					Owned:    []csvv1alpha1.CRDDescription{},
					Required: []csvv1alpha1.CRDDescription{},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset()
	store := CustomResourceCatalogStore{Client: fakeClient, Namespace: "alm-coreos-tests"}

	actualEntry, err := store.Store(manifest, &csv, ownerRefs)
	require.NoError(t, err)

	expectedActions := []core.Action{
		core.NewGetAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry.GetName(),
		),
		core.NewCreateAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry,
		),
		core.NewGetAction(
			schema.GroupVersionResource{Group: "app.coreos.com", Version: "v1alpha1", Resource: "uicatalogentry-v1s"},
			expectedEntry.GetNamespace(),
			expectedEntry.GetName(),
		),
	}

	returnEntry, err := fakeClient.UicatalogentryV1alpha1().UICatalogEntries(expectedEntry.GetNamespace()).Get(expectedEntry.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
	RequireSemanticEqual(t, returnEntry, actualEntry)
	RequireActions(t, expectedActions, fakeClient.Actions())
}

//func TestCustomResourceCatalogStoreSync(t *testing.T) {
//	src := NewInMem()
//	source := catsrcv1alpha1.CatalogSource{}
//
//	testCSVNameA := "MockServiceNameA-v1"
//	testCSVVersionA1 := "0.2.4+alpha"
//	testPackageA := v1alpha1.PackageManifest{
//		PackageName: "MockServiceA",
//		Channels: []v1alpha1.PackageChannel{
//			{
//				Name:           "alpha",
//				CurrentCSVName: testCSVNameA,
//			},
//		},
//	}
//
//	testCSVNameB := "MockServiceNameB-v1"
//	testCSVVersionB1 := "1.0.1"
//	testPackageB := v1alpha1.PackageManifest{
//		PackageName: "MockServiceB",
//		Channels: []v1alpha1.PackageChannel{
//			{
//				Name:           "alpha",
//				CurrentCSVName: testCSVNameB,
//			},
//		},
//	}
//
//	testCSVA1 := createCSV(testCSVNameA, testCSVVersionA1, "", []string{})
//	testCSVB1 := createCSV(testCSVNameB, testCSVVersionB1, "", []string{})
//	src.AddOrReplaceService(testCSVA1)
//	src.AddOrReplaceService(testCSVB1)
//	require.NoError(t, src.addPackageManifest(testPackageA))
//	require.NoError(t, src.addPackageManifest(testPackageB))
//
//	storeResults := []struct {
//		ResultA1 *v1alpha1.UICatalogEntry
//		ErrorA1  error
//
//		ResultB1 *v1alpha1.UICatalogEntry
//		ErrorB1  error
//
//		ExpectedStatus         string
//		ExpectedServicesSynced int
//	}{
//		{
//			&v1alpha1.UICatalogEntry{ObjectMeta: metav1.ObjectMeta{Name: testCSVNameA}}, nil,
//			&v1alpha1.UICatalogEntry{ObjectMeta: metav1.ObjectMeta{Name: testCSVNameB}}, nil,
//			"success", 2,
//		},
//		{
//			&v1alpha1.UICatalogEntry{ObjectMeta: metav1.ObjectMeta{Name: testCSVNameA}}, nil,
//			nil, errors.New("test error"),
//			"error", 1,
//		},
//		{
//			nil, errors.New("test error1"),
//			&v1alpha1.UICatalogEntry{ObjectMeta: metav1.ObjectMeta{Name: testCSVNameB}}, nil,
//			"error", 1,
//		},
//	}
//
//	for _, res := range storeResults {
//		fakeClient := fake.NewSimpleClientset()
//		store := CustomResourceCatalogStore{Client: fakeClient, Namespace: "alm-coreos-tests"}
//
//		fakeClient.ListEntriesReturns(nil, nil)
//		fakeClient.DeleteReturns(nil)
//
//		fakeClient.UpdateEntryReturnsOnCall(0, res.ResultA1, res.ErrorA1)
//		fakeClient.UpdateEntryReturnsOnCall(1, res.ResultB1, res.ErrorB1)
//
//		entries, err := store.Sync(src, &source)
//		require.Equal(t, res.ExpectedServicesSynced, len(entries))
//		require.Equal(t, res.ExpectedStatus, store.LastAttemptedSync.Status)
//		require.NoError(t, err)
//		require.Equal(t, 2, fakeClient.UpdateEntryCallCount())
//	}
//}

//func TestPruneUICatalogEntries(t *testing.T) {
//	source := catsrcv1alpha1.CatalogSource{
//		TypeMeta: metav1.TypeMeta{
//			Kind:       catsrcv1alpha1.CatalogSourceKind,
//			APIVersion: catsrcv1alpha1.CatalogSourceCRDAPIVersion,
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "test-source",
//		},
//	}
//	ownerRefs := []metav1.OwnerReference{
//		*metav1.NewControllerRef(&source, source.GroupVersionKind()),
//	}
//
//	type catalogState struct {
//		csvs     []*csvv1alpha1.ClusterServiceVersion
//		crds     []*v1beta1.CustomResourceDefinition
//		packages []*v1alpha1.PackageManifest
//	}
//	type clusterState struct {
//		entries []*v1alpha1.UICatalogEntry
//	}
//	type outState struct {
//		createdOrUpdated []*v1alpha1.UICatalogEntry
//		pruned           []*v1alpha1.UICatalogEntry
//	}
//	tests := []struct {
//		in          catalogState
//		out         outState
//		state       clusterState
//		err         error
//		description string
//	}{
//		{
//			state: clusterState{
//				entries: []*v1alpha1.UICatalogEntry{},
//			},
//			in: catalogState{
//				csvs: []*csvv1alpha1.ClusterServiceVersion{
//					makeCSV("service1", "1.0.0", makeCRDs("owned1"), makeCRDs("required1")),
//				},
//				crds: makeCRDs("owned1", "required1"),
//				packages: []*v1alpha1.PackageManifest{
//					{
//						PackageName: "service",
//						Channels: []v1alpha1.PackageChannel{
//							{
//								Name:           "alpha",
//								CurrentCSVName: "service1",
//							},
//						},
//					},
//				},
//			},
//			out: outState{
//				createdOrUpdated: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service1", "1.0.0", makeCRDs("owned1"), makeCRDs("required1")),
//						v1alpha1.PackageManifest{
//							PackageName: "service",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service1",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//			},
//			description: "NoExistingEntries",
//		},
//		{
//			state: clusterState{
//				entries: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service1", "1.0.0", makeCRDs("owned1"), makeCRDs("required1")),
//						v1alpha1.PackageManifest{
//							PackageName: "service",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service1",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//			},
//			in: catalogState{
//				csvs: []*csvv1alpha1.ClusterServiceVersion{
//					makeCSV("service2", "1.0.2", makeCRDs("owned2"), makeCRDs("required2")),
//				},
//				crds: makeCRDs("owned2", "required2"),
//				packages: []*v1alpha1.PackageManifest{
//					{
//						PackageName: "service",
//						Channels: []v1alpha1.PackageChannel{
//							{
//								Name:           "alpha",
//								CurrentCSVName: "service2",
//							},
//						},
//					},
//				},
//			},
//			out: outState{
//				createdOrUpdated: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service2", "1.0.2", makeCRDs("owned2"), makeCRDs("required2")),
//						v1alpha1.PackageManifest{
//							PackageName: "service",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service2",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//			},
//			description: "UpdateExistingEntries",
//		},
//		{
//			state: clusterState{
//				entries: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service1", "1.0.0", makeCRDs("owned1"), makeCRDs("required1")),
//						v1alpha1.PackageManifest{
//							PackageName: "service",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service1",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//			},
//			in: catalogState{
//				csvs: []*csvv1alpha1.ClusterServiceVersion{
//					makeCSV("service2", "1.0.2", makeCRDs("owned2"), makeCRDs("required2")),
//				},
//				crds: makeCRDs("owned2", "required2"),
//				packages: []*v1alpha1.PackageManifest{
//					{
//						PackageName: "service2",
//						Channels: []v1alpha1.PackageChannel{
//							{
//								Name:           "alpha",
//								CurrentCSVName: "service2",
//							},
//						},
//					},
//				},
//			},
//			out: outState{
//				createdOrUpdated: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service2", "1.0.2", makeCRDs("owned2"), makeCRDs("required2")),
//						v1alpha1.PackageManifest{
//							PackageName: "service2",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service2",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//				pruned: []*v1alpha1.UICatalogEntry{
//					uiCatalogEntry(
//						makeCSV("service1", "1.0.0", makeCRDs("owned1"), makeCRDs("required1")),
//						v1alpha1.PackageManifest{
//							PackageName: "service",
//							Channels: []v1alpha1.PackageChannel{
//								{
//									Name:           "alpha",
//									CurrentCSVName: "service1",
//								},
//							},
//						},
//						ownerRefs,
//					),
//				},
//			},
//			description: "PruneExistingAndCreateEntries",
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.description, func(t *testing.T) {
//
//			// configure store and cluster
//			store := CustomResourceCatalogStore{Namespace: "alm-coreos-tests"}
//			src := NewInMem()
//			fakeClient := new(clientfakes.FakeUICatalogEntryInterface)
//			store.Client = fakeClient
//
//			for _, crd := range tt.in.crds {
//				require.NoError(t, src.SetCRDDefinition(*crd))
//			}
//			for _, csv := range tt.in.csvs {
//				src.AddOrReplaceService(*csv)
//			}
//			for _, manifest := range tt.in.packages {
//				require.NoError(t, src.addPackageManifest(*manifest))
//			}
//			fakeClient.ListEntriesReturns(&v1alpha1.UICatalogEntryList{Items: tt.state.entries}, nil)
//			for i, entry := range tt.out.createdOrUpdated {
//				fakeClient.UpdateEntryReturnsOnCall(i, entry, nil)
//			}
//			fakeClient.DeleteReturns(nil)
//
//			// sync source with cluster state
//			store.Sync(src, &source)
//
//			// verify the right entries were created/updated
//			require.Equal(t, len(tt.out.createdOrUpdated), fakeClient.UpdateEntryCallCount())
//			for i, entry := range tt.out.createdOrUpdated {
//				require.EqualValues(t, entry, fakeClient.UpdateEntryArgsForCall(i))
//			}
//			for i, entry := range tt.out.pruned {
//				_, prunedName, _ := fakeClient.DeleteArgsForCall(i)
//				require.EqualValues(t, entry.Name, prunedName)
//			}
//		})
//	}
//}