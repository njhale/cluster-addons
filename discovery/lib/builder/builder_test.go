package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestGjson(t *testing.T) {
	crd := `{"apiVersion":"apiextensions.k8s.io/v1beta1","kind":"CustomResourceDefinition","metadata":{"name":"etcdrestores.etcd.database.coreos.com"},"spec":{"group":"etcd.database.coreos.com","names":{"kind":"EtcdRestore","listKind":"EtcdRestoreList","plural":"etcdrestores","singular":"etcdrestore"},"scope":"Namespaced",    "versions": [
      {
        "name": "v1beta1",
        "served": true,
        "storage": true,
        "schema": {
          "openAPIV3Schema": {
            "type": "object",
            "properties": {
              "host": {
                "type": "string"
              },
              "port": {
                "type": "string"
              }
            }
          }
        }
      },
      {
        "name": "v1",
        "served": true,
        "storage": false,
        "schema": {
          "openAPIV3Schema": {
            "type": "object",
            "properties": {
              "host": {
                "type": "string"
              },
              "port": {
                "type": "string"
              }
            }
          }
        }
      }
    ]}}`

	// basic queries
	group := gjson.Get(crd, "spec.group")
	require.Equal(t, "\"etcd.database.coreos.com\"", group.Raw)
	versions := gjson.Get(crd, "spec.versions.#.name")
	require.Equal(t, "[\"v1beta1\",\"v1\"]", versions.Raw)
	kind := gjson.Get(crd, "spec.names.kind")
	require.Equal(t, "\"EtcdRestore\"", kind.Raw)
	plural := gjson.Get(crd, "spec.names.plural")
	require.Equal(t, "\"etcdrestores\"", plural.Raw)

	// basic setting
	out, err := sjson.SetRaw("", "group", group.Raw)
	require.NoError(t, err)
	require.Equal(t, "{\"group\":\"etcd.database.coreos.com\"}", out)
}

func TestBuildObject(t *testing.T) {
	crd := `{"apiVersion":"apiextensions.k8s.io/v1beta1","kind":"CustomResourceDefinition","metadata":{"name":"etcdrestores.etcd.database.coreos.com"},"spec":{"group":"etcd.database.coreos.com","names":{"kind":"EtcdRestore","listKind":"EtcdRestoreList","plural":"etcdrestores","singular":"etcdrestore"},"scope":"Namespaced",    "versions": [
      {
        "name": "v1beta1",
        "served": true,
        "storage": true,
        "schema": {
          "openAPIV3Schema": {
            "type": "object",
            "properties": {
              "host": {
                "type": "string"
              },
              "port": {
                "type": "string"
              }
            }
          }
        }
      },
      {
        "name": "v1",
        "served": true,
        "storage": false,
        "schema": {
          "openAPIV3Schema": {
            "type": "object",
            "properties": {
              "host": {
                "type": "string"
              },
              "port": {
                "type": "string"
              }
            }
          }
        }
      }
    ]}}`

	// collect into an array
	outset, err := BuildObject(`
    {"group": "spec.group", "versions": ["spec.versions.#.name"], "kind": "spec.names.kind", "plural": "spec.names.plural"}
     `, crd)
	require.NoError(t, err)
	require.Equal(t, "{\"plural\":\"etcdrestores\",\"kind\":\"EtcdRestore\",\"versions\":[\"v1beta1\",\"v1\"],\"group\":\"etcd.database.coreos.com\"}", outset[0])

	// one object per array entry
	outset, err = BuildObject(`
    {"group": "spec.group", "version": "spec.versions.#.name", "kind": "spec.names.kind", "plural": "spec.names.plural"}
     `, crd)
	require.NoError(t, err)
	require.Equal(t, []string{
		"{\"plural\":\"etcdrestores\",\"kind\":\"EtcdRestore\",\"version\":\"v1beta1\",\"group\":\"etcd.database.coreos.com\"}",
		"{\"plural\":\"etcdrestores\",\"kind\":\"EtcdRestore\",\"version\":\"v1\",\"group\":\"etcd.database.coreos.com\"}",
	}, outset)

	namespace := `{"apiVersion":"v1","kind":"Namespace","metadata":{"annotations":{"discovery.addons.x-k8s.io/my.friendly.id":"{\"namespace\": \"name\"}"},"creationTimestamp":"2020-02-17T23:56:02Z","labels":{"discovery.addons.x-k8s.io/ghost-7v5lw":""},"name":"ns-nr8wl","resourceVersion":"54","selfLink":"/api/v1/namespaces/ns-nr8wl","uid":"a85bedf8-5e03-43ba-9d42-1c9cb40c80e0"},"spec":{"finalizers":["kubernetes"]},"status":{"phase":"Active"}}`
	extractName, err := BuildObject(`{"namespace": "metadata.name"}`, namespace)
	require.NoError(t, err)
	require.Equal(t, `{"namespace":"ns-nr8wl"}`, extractName[0])
}
