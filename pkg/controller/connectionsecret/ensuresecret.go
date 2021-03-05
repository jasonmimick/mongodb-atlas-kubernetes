package connectionsecret

import (
	"context"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mongodb/mongodb-atlas-kubernetes/pkg/util/kube"
)

const (
	ProjectLabelKey string = "atlas.mongodb.com/project-id"
	ClusterLabelKey string = "atlas.mongodb.com/cluster-name"

	connectionSecretStdKey    string = "connectionString.standard"
	connectionSecretStdSrvKey string = "connectionString.standardSrv"
	userNameKey               string = "username"
	passwordKey               string = "password"
)

type ConnectionData struct {
	dbUserName, connURL, srvConnURL, password string
}

// Ensure creates or updates the connection Secret for the specific cluster and db user.
func Ensure(client client.Client, namespace, projectName, projectID, clusterName string, data ConnectionData) error {
	var getError error
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      formatSecretName(projectName, clusterName, data.dbUserName),
		Namespace: namespace,
	}}
	if getError = client.Get(context.Background(), kube.ObjectKeyFromObject(s), s); getError != nil && !apiErrors.IsNotFound(getError) {
		return getError
	}
	if err := fillSecret(s, projectID, clusterName, data); err != nil {
		return err
	}
	if getError != nil {
		// Creating
		return client.Create(context.Background(), s)
	}

	return client.Update(context.Background(), s)
}

func fillSecret(secret *corev1.Secret, projectID string, clusterName string, data ConnectionData) error {
	var connURL, srvConnURL string
	var err error
	if connURL, err = addCredentialsToConnectionURL(data.connURL, data.dbUserName, data.password); err != nil {
		return err
	}
	if srvConnURL, err = addCredentialsToConnectionURL(data.srvConnURL, data.dbUserName, data.password); err != nil {
		return err
	}

	secret.Labels = map[string]string{ProjectLabelKey: projectID, ClusterLabelKey: kube.NormalizeLabelValue(clusterName)}

	secret.Data = map[string][]byte{
		connectionSecretStdKey:    []byte(connURL),
		connectionSecretStdSrvKey: []byte(srvConnURL),
		userNameKey:               []byte(data.dbUserName),
		passwordKey:               []byte(data.password),
	}
	return nil
}

func formatSecretName(projectName, clusterName, dbUserName string) string {
	name := fmt.Sprintf("%s-%s-%s",
		kube.NormalizeIdentifier(projectName),
		kube.NormalizeIdentifier(clusterName),
		kube.NormalizeIdentifier(dbUserName))
	return kube.NormalizeIdentifier(name)
}

func addCredentialsToConnectionURL(connURL, userName, password string) (string, error) {
	cs, err := url.Parse(connURL)
	if err != nil {
		return "", err
	}
	cs.User = url.UserPassword(userName, password)
	return cs.String(), nil
}