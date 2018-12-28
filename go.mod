module github.com/steigr/csi-zfs

require (
	github.com/container-storage-interface/spec v1.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/kubernetes-csi/drivers v1.0.1
	github.com/pborman/uuid v1.2.0
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.2.0 // indirect
	github.com/steigr/zfsd v0.0.1
	github.com/stretchr/testify v1.2.2 // indirect
	golang.org/x/net v0.0.0-20181217023233-e147a9138326
	google.golang.org/grpc v1.17.0
	k8s.io/apimachinery v0.0.0-20181222072933-b814ad55d7c5 // indirect
	k8s.io/klog v0.1.0 // indirect
	k8s.io/kubernetes v1.13.1
	k8s.io/utils v0.0.0-20181115163542-0d26856f57b3 // indirect
)

replace github.com/container-storage-interface/spec => ../../../github.com/container-storage-interface/spec
