/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package zfs

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	pb "github.com/steigr/zfsd/pkg/proto"
	"github.com/steigr/zfsd/pkg/util"
	"strconv"
	"strings"
)

const (
	kib    int64 = 1024
	mib    int64 = kib * 1024
	gib    int64 = mib * 1024
	gib100 int64 = gib * 100
	tib    int64 = gib * 1024
	tib100 int64 = tib * 100

	defaultVolumePropertyName        = "storage.k8s.io:volume-default"
	rootVolumePropertyName           = "storage.k8s.io:volume-root"
	volumeTagsProperty               = "storage.k8s.io:tags"
	volumeReservationProperty        = "reservation"
	volumeMountpointProperty         = "mountpoint"
	volumeIdProperty                 = "storage.k8s.io:volume-id"
	volumeLimitProperty              = "quota"
	volumePropertyTuningLogbias      = "logbias"
	volumePropertyTuningPrimarycache = "primarycache"
	volumePropertyTuningRecordsize   = "recordsize"
)

var (
	userProperties     = []string{defaultVolumePropertyName, rootVolumePropertyName}
	volumeIdProperties = []string{volumeIdProperty}
)

type zfs struct {
	driver *csicommon.CSIDriver

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer

	cap   []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability

	Pools    pb.PoolClient
	Datasets pb.DatasetClient

	ProvisionRoot string
}

type zfsVolume struct {
	Name             string
	Tags             []string
	RequiredCapacity int64
	MaximumCapacity  int64
	Mountpoint       string
	VolumeId         string
	Logbias          string
	Recordsize       string
	Primarycache     string
}

var (
	zfsDriver     *zfs
	vendorVersion = "dev"
)

func GetDriver() *zfs {
	zfsDriver = &zfs{}
	return zfsDriver
}

func NewIdentityServer(d *csicommon.CSIDriver) *identityServer {
	return &identityServer{
		DefaultIdentityServer: csicommon.NewDefaultIdentityServer(d),
	}
}

func NewControllerServer(d *csicommon.CSIDriver) *controllerServer {
	return &controllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(d),
	}
}

func NewNodeServer(d *csicommon.CSIDriver) *nodeServer {
	return &nodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d),
	}
}

func (z *zfs) Run(driverName, nodeID, endpoint string) {
	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	// Initialize default library driver
	z.driver = csicommon.NewCSIDriver(driverName, vendorVersion, nodeID)
	if z.driver == nil {
		glog.Fatalln("Failed to initialize CSI Driver.")
	}
	z.driver.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
			csi.ControllerServiceCapability_RPC_GET_CAPACITY,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
			csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
			csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_READONLY,
			csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		})
	z.driver.AddVolumeCapabilityAccessModes(
		[]csi.VolumeCapability_AccessMode_Mode{
			csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		})

	// Create GRPC servers
	z.ids = NewIdentityServer(z.driver)
	z.ns = NewNodeServer(z.driver)
	z.cs = NewControllerServer(z.driver)

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(endpoint, z.ids, z.cs, z.ns)
	s.Wait()
}

func MountpointFor(volumeId string) string {
	return zfsDriver.ProvisionRoot + volumeId
}

func NewDatasetClient(endpoint string) (cl pb.DatasetClient, err error) {
	cl, err = util.NewDatasetClient(endpoint)
	if err != nil {
		return nil, err
	}
	glog.V(9).Info("attached to dataset service")
	return cl, nil
}

func NewPoolClient(endpoint string) (cl pb.PoolClient, err error) {
	cl, err = util.NewPoolClient(endpoint)
	if err != nil {
		return nil, err
	}
	glog.V(9).Info("attached to pool service")
	return cl, nil
}

func VolumeCreate(volume *zfsVolume) error {
	// Ensure quota is always set
	if volume.MaximumCapacity < volume.RequiredCapacity {
		volume.MaximumCapacity = volume.RequiredCapacity
	}

	request := &pb.CreateDatasetRequest{
		Dataset: &pb.DatasetT{
			Path: volume.Name,
			Properties: []*pb.Property{
				{
					Key:   volumeLimitProperty,
					Value: strconv.Itoa(int(volume.MaximumCapacity)),
				},
				{
					Key:   volumeReservationProperty,
					Value: strconv.Itoa(int(volume.RequiredCapacity)),
				},
				{
					Key:   volumeTagsProperty,
					Value: strings.Join(volume.Tags, ","),
				},
				{
					Key:   volumeMountpointProperty,
					Value: volume.Mountpoint,
				},
				{
					Key:   volumeIdProperty,
					Value: volume.VolumeId,
				},
				{
					Key:   volumePropertyTuningLogbias,
					Value: volume.Logbias,
				},
				{
					Key:   volumePropertyTuningPrimarycache,
					Value: volume.Primarycache,
				},
				{
					Key:   volumePropertyTuningRecordsize,
					Value: volume.Recordsize,
				},
			},
		},
	}
	_, err := zfsDriver.Datasets.Create(context.TODO(), request)
	return err
}

func VolumeDelete(volume *zfsVolume) error {
	request := &pb.DeleteDatasetRequest{
		Path: volume.Name,
	}
	_, err := zfsDriver.Datasets.Delete(context.TODO(), request)
	return err
}

func VolumeUpdate(volume *zfsVolume, properties map[string]string) error {
	request := &pb.UpdateDatasetRequest{
		Dataset: &pb.DatasetT{
			Path: volume.Name,
		},
	}
	for key, value := range properties {
		request.Dataset.Properties = append(request.Dataset.Properties, &pb.Property{
			Key:   key,
			Value: value,
		})
	}
	_, err := zfsDriver.Datasets.Update(context.TODO(), request)
	return err
}

func VolumeMount() error {
	return nil
}

func VolumeUnmount() error {
	return nil
}

func DatasetGetDefault() (name string, err error) {
	name = ""
	datasets := mustDatasets()
	if len(datasets) < 1 {
		return "", fmt.Errorf("no volumes found")
	}
	for _, dataset := range mustDatasets() {
		if name, err = getPropertyOfDataset(dataset, defaultVolumePropertyName); err != nil {
			err = nil
			continue
		}
		if len(name) > 0 {
			break
		}
	}
	if len(name) == 0 {
		glog.Warningf("assign property '%s' to the root-volume of at least one pool", defaultVolumePropertyName)
		name = mustDatasets()[0].Path
	}
	glog.V(9).Info("Using ", name, " as default pool")
	return name, nil
}

func DatasetGetRoot(pool string) (name string, err error) {
	name = ""
	datasets := mustDatasets()
	if len(datasets) < 1 {
		return "", fmt.Errorf("no volumes found")
	}
	for _, dataset := range mustDatasets() {
		if !strings.HasPrefix(dataset.Path, pool) {
			continue
		}
		if name, err = getPropertyOfDataset(dataset, rootVolumePropertyName); err != nil {
			err = nil
			continue
		}
		if name == "-" {
			name = ""
		}
		if len(name) > 0 {
			break
		}
	}
	if len(name) == 0 {
		glog.Warningf("assign property '%s' to the root-volume in each pool", rootVolumePropertyName)
		name = mustDatasets()[0].Path
	}
	glog.V(9).Info("Using ", name, " as default volume")
	return name, nil
}

// mustDatasets only returns the root volumes of each pool
func mustDatasets() (datasets []*pb.DatasetT) {
	reply, err := zfsDriver.Datasets.List(context.TODO(), &pb.ListDatasetRequest{
		UserProperties: userProperties,
	})
	if err != nil {
		glog.Fatalf("cannot list datasets")
		return nil
	}
	return reply.Datasets
}

func getPropertyOfDataset(ds *pb.DatasetT, propertyName string) (string, error) {
	for _, property := range ds.Properties {
		if property.Key == propertyName {
			return property.Value, nil
		}
	}
	return "", fmt.Errorf("property %s not defined in volume %s", propertyName, ds.Path)
}

func GetVolumePathById(volumeId string) (name string, err error) {
	reply, err := zfsDriver.Datasets.List(context.TODO(), &pb.ListDatasetRequest{
		UserProperties: volumeIdProperties,
	})
	var result *pb.DatasetT
	for _, ds := range reply.Datasets {
		result, err = filterDatasetByProperty(ds, volumeIdProperty, volumeId)
	}
	if err != nil {
		return "", err
	}
	return result.Path, nil
}

func filterDatasetByProperty(in *pb.DatasetT, property, value string) (out *pb.DatasetT, err error) {
	glog.V(9).Infof("Checking dataset %s for %s==%s", in.Path, property, value)
	for _, prop := range in.Properties {
		if prop.Key == property && prop.Value == value {
			glog.V(9).Infof("Dataset %s matches", in.Path)
			return in, nil
		}
	}
	if len(in.Datasets) > 0 {
		for _, ds := range in.Datasets {
			return filterDatasetByProperty(ds, property, value)
		}
	}
	return nil, fmt.Errorf("dataset with %s=%s not found", property, value)
}
