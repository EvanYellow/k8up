package v1

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"

	"github.com/k8up-io/k8up/operator/cfg"
)

type (
	// Backend allows configuring several backend implementations.
	// It is expected that users only configure one storage type.
	Backend struct {
		// RepoPasswordSecretRef references a secret key to look up the restic repository password
		RepoPasswordSecretRef string          `json:"repoPasswordSecretRef,omitempty"`
		ResticOptions         string          `json:"resticOptions,omitempty"`
		Local                 *LocalSpec      `json:"local,omitempty"`
		S3                    *S3Spec         `json:"s3,omitempty"`
		GCS                   *GCSSpec        `json:"gcs,omitempty"`
		Azure                 *AzureSpec      `json:"azure,omitempty"`
		Swift                 *SwiftSpec      `json:"swift,omitempty"`
		B2                    *B2Spec         `json:"b2,omitempty"`
		Rest                  *RestServerSpec `json:"rest,omitempty"`
	}

	// +k8s:deepcopy-gen=false

	// BackendInterface represents a Backend for internal use.
	BackendInterface interface {
		EnvVars(vars map[string]string) map[string]string
		String() string
	}
)

// GetCredentialEnv will return a map containing the credentials for the given backend.
func (in *Backend) GetCredentialEnv() map[string]string {
	vars := make(map[string]string)

	//if in.RepoPasswordSecretRef != nil {
	//	vars[cfg.ResticPasswordEnvName] = &corev1.EnvVarSource{
	//		SecretKeyRef: in.RepoPasswordSecretRef,
	//	}
	//}

	for _, backend := range in.getSupportedBackends() {
		if IsNil(backend) {
			continue
		}
		return backend.EnvVars(vars)
	}

	return nil
}

// GetResticPasswords will return a map containing the credentials for the given backend.
func (in *Backend) GetResticPasswords() string {
	if in.RepoPasswordSecretRef != "" {
		return in.RepoPasswordSecretRef
	}
	return ""
}

// GetResticOptions will return a map containing the credentials for the given backend.
func (in *Backend) GetResticOptions() string {
	if in.ResticOptions != "" {
		return in.ResticOptions
	}
	return ""
}

// String returns the string representation of the repository. If no repo is
// defined it'll return empty string.
func (in *Backend) String() string {

	for _, backend := range in.getSupportedBackends() {
		if IsNil(backend) {
			continue
		}
		return backend.String()
	}
	return ""

}

// IsBackendEqualTo returns true if the restic repository string is equal to the other's string.
// If other is nil, it returns false.
func (in *Backend) IsBackendEqualTo(other *Backend) bool {
	if other == nil {
		return false
	}
	return in.String() == other.String()
}

func (in *Backend) getSupportedBackends() []BackendInterface {
	return []BackendInterface{in.Azure, in.B2, in.GCS, in.Local, in.Rest, in.S3, in.Swift}
}

// IsNil returns true if the given value is nil using reflect.
func IsNil(v interface{}) bool {
	// Unfortunately "v == nil" doesn't work with Interfaces, since they are tuples containing type and value.
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

func addEnvVarFromSecret(vars map[string]string, key string, ref string) {
	//if ref != nil {
	//	vars[key] = &corev1.EnvVarSource{
	//		SecretKeyRef: ref,
	//	}
	//}
	if ref != "" {
		vars[key] = ref
	}
}

type LocalSpec struct {
	MountPath string `json:"mountPath,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *LocalSpec) EnvVars(vars map[string]string) map[string]string {
	return vars
}

// String returns the mountpath.
func (in *LocalSpec) String() string {
	return in.MountPath
}

type S3Spec struct {
	Endpoint                 string `json:"endpoint,omitempty"`
	Bucket                   string `json:"bucket,omitempty"`
	AccessKeyIDSecretRef     string `json:"accessKeyIDSecretRef,omitempty"`
	SecretAccessKeySecretRef string `json:"secretAccessKeySecretRef,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *S3Spec) EnvVars(vars map[string]string) map[string]string {
	addEnvVarFromSecret(vars, cfg.AwsAccessKeyIDEnvName, in.AccessKeyIDSecretRef)
	addEnvVarFromSecret(vars, cfg.AwsSecretAccessKeyEnvName, in.SecretAccessKeySecretRef)
	return vars
}

// String returns "s3:endpoint/bucket".
// If endpoint or bucket are empty, it uses their global setting accordingly.
func (in *S3Spec) String() string {
	endpoint := cfg.Config.GlobalS3Endpoint
	if in.Endpoint != "" {
		endpoint = in.Endpoint
	}

	bucket := cfg.Config.GlobalS3Bucket
	if in.Bucket != "" {
		bucket = in.Bucket
	}

	return fmt.Sprintf("s3:%v/%v", endpoint, bucket)
}

// RestoreEnvVars returns the env vars for this backend when using Restore jobs.
func (in *S3Spec) RestoreEnvVars() map[string]*corev1.EnvVar {
	vars := make(map[string]*corev1.EnvVar)
	if in.AccessKeyIDSecretRef != "" {
		//vars[cfg.RestoreS3AccessKeyIDEnvName] = &corev1.EnvVar{
		//	ValueFrom: &corev1.EnvVarSource{
		//		SecretKeyRef: in.AccessKeyIDSecretRef,
		//	},
		//}
		vars[cfg.RestoreS3AccessKeyIDEnvName] = &corev1.EnvVar{
			Value: in.AccessKeyIDSecretRef,
		}
	} else {
		vars[cfg.RestoreS3AccessKeyIDEnvName] = &corev1.EnvVar{
			Value: cfg.Config.GlobalRestoreS3AccessKey,
		}
	}

	if in.SecretAccessKeySecretRef != "" {
		//vars[cfg.RestoreS3SecretAccessKeyEnvName] = &corev1.EnvVar{
		//	ValueFrom: &corev1.EnvVarSource{
		//		SecretKeyRef: in.SecretAccessKeySecretRef,
		//	},
		//}
		vars[cfg.RestoreS3SecretAccessKeyEnvName] = &corev1.EnvVar{
			Value: in.SecretAccessKeySecretRef,
		}
	} else {
		vars[cfg.RestoreS3SecretAccessKeyEnvName] = &corev1.EnvVar{
			Value: cfg.Config.GlobalRestoreS3SecretAccessKey,
		}
	}

	bucket := in.Bucket
	endpoint := in.Endpoint
	if bucket == "" {
		bucket = cfg.Config.GlobalRestoreS3Bucket
	}
	if endpoint == "" {
		endpoint = cfg.Config.GlobalRestoreS3Endpoint
	}

	vars[cfg.RestoreS3EndpointEnvName] = &corev1.EnvVar{
		Value: fmt.Sprintf("%v/%v", endpoint, bucket),
	}

	return vars
}

type GCSSpec struct {
	Bucket               string `json:"bucket,omitempty"`
	ProjectIDSecretRef   string `json:"projectIDSecretRef,omitempty"`
	AccessTokenSecretRef string `json:"accessTokenSecretRef,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *GCSSpec) EnvVars(vars map[string]string) map[string]string {
	addEnvVarFromSecret(vars, cfg.GcsProjectIDEnvName, in.ProjectIDSecretRef)
	addEnvVarFromSecret(vars, cfg.GcsAccessTokenEnvName, in.AccessTokenSecretRef)
	return vars

}

// String returns "gs:bucket:/"
func (in *GCSSpec) String() string {
	return fmt.Sprintf("gs:%s:/", in.Bucket)
}

type AzureSpec struct {
	Container            string `json:"container,omitempty"`
	AccountNameSecretRef string `json:"accountNameSecretRef,omitempty"`
	AccountKeySecretRef  string `json:"accountKeySecretRef,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *AzureSpec) EnvVars(vars map[string]string) map[string]string {
	addEnvVarFromSecret(vars, cfg.AzureAccountKeyEnvName, in.AccountKeySecretRef)
	addEnvVarFromSecret(vars, cfg.AzureAccountEnvName, in.AccountNameSecretRef)
	return vars
}

// String returns "azure:container:/"
func (in *AzureSpec) String() string {
	return fmt.Sprintf("azure:%s:/", in.Container)
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Path      string `json:"path,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *SwiftSpec) EnvVars(vars map[string]string) map[string]string {
	return vars
}

// String returns "swift:container:path"
func (in *SwiftSpec) String() string {
	return fmt.Sprintf("swift:%s:%s", in.Container, in.Path)
}

type B2Spec struct {
	Bucket              string `json:"bucket,omitempty"`
	Path                string `json:"path,omitempty"`
	AccountIDSecretRef  string `json:"accountIDSecretRef,omitempty"`
	AccountKeySecretRef string `json:"accountKeySecretRef,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *B2Spec) EnvVars(vars map[string]string) map[string]string {
	addEnvVarFromSecret(vars, cfg.B2AccountIDEnvName, in.AccountIDSecretRef)
	addEnvVarFromSecret(vars, cfg.B2AccountKeyEnvName, in.AccountKeySecretRef)
	return vars
}

// String returns "b2:bucket:path"
func (in *B2Spec) String() string {
	return fmt.Sprintf("b2:%s:%s", in.Bucket, in.Path)
}

type RestServerSpec struct {
	URL               string `json:"url,omitempty"`
	UserSecretRef     string `json:"userSecretRef,omitempty"`
	PasswordSecretReg string `json:"passwordSecretReg,omitempty"`
}

// EnvVars returns the env vars for this backend.
func (in *RestServerSpec) EnvVars(vars map[string]string) map[string]string {
	addEnvVarFromSecret(vars, cfg.RestPasswordEnvName, in.PasswordSecretReg)
	addEnvVarFromSecret(vars, cfg.RestUserEnvName, in.UserSecretRef)
	return vars
}

// String returns "rest:URL"
func (in *RestServerSpec) String() string {
	return fmt.Sprintf("rest:%s", in.URL)
}
