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
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	deviceID           = "deviceID"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.V(9).Info("CreateVolume()")
	glog.V(9).Info(spew.Sdump(req))

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	if len(req.GetName()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	caps := req.GetVolumeCapabilities()
	if caps == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}
	// TODO: implement block support via vdevs
	for _, cap := range caps {
		if cap.GetBlock() != nil {
			return nil, status.Error(codes.Unimplemented, "Block Volume not implemented")
		}
	}
	volumeID := uuid.NewUUID().String()
	var pool, logbias, primarycache, recordsize string
	var isset bool

	if pool, isset = req.Parameters["pool"]; !isset {
		var err error
		pool,err = DatasetGetDefault()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "ZFS default pool undetected")
		}
	}

	if logbias, isset = req.Parameters["logbias"]; !isset {
		logbias = "latency"
	}

	if primarycache, isset = req.Parameters["primarycache"]; !isset {
		primarycache = "all"
	}

	if recordsize, isset = req.Parameters["recordsize"]; !isset {
		recordsize = "127K"
	}

	parentVolume,err := DatasetGetRoot(pool)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "ZFS root volume in "+ pool + " undefined")
	}
	volumeName := parentVolume + "/pvc-" + volumeID

	if err = VolumeCreate(&zfsVolume{
		Name: volumeName,
		RequiredCapacity: req.GetCapacityRange().GetRequiredBytes(),
		MaximumCapacity: req.GetCapacityRange().GetLimitBytes(),
		Mountpoint: MountpointFor(volumeID),
		VolumeId: volumeID,
		Logbias: logbias,
		Primarycache: primarycache,
		Recordsize: recordsize,

	}); err != nil{
		return nil, err
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			VolumeContext: req.GetParameters(),
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.V(9).Info("DeleteVolume()")

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid delete volume req: %v", req)
		return nil, err
	}

	if ds,err := GetVolumePathById(req.VolumeId); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Volume with ID %s not found: %v",req.VolumeId,err))
	} else {
		volume := &zfsVolume{
			Name: ds,
		}
		unsetMountpointProperties := map[string]string{
			"mountpoint": "none",
		}
		if err = VolumeUpdate(volume,unsetMountpointProperties); err != nil {
			return nil, err
		}
		if err = VolumeDelete(volume); err != nil{
			return nil, err
		}
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	glog.V(9).Info("ValidateVolumeCapabilities()")
	return cs.DefaultControllerServer.ValidateVolumeCapabilities(ctx, req)
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	glog.V(9).Info("CreateSnapshot()")
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		glog.V(3).Infof("invalid create snapshot req: %v", req)
		return nil, err
	}

	return &csi.CreateSnapshotResponse{}, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	glog.V(9).Info("DeleteSnapshot()")
	// Check arguments
	if len(req.GetSnapshotId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot ID missing in request")
	}

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		glog.V(3).Infof("invalid delete snapshot req: %v", req)
		return nil, err
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	glog.V(9).Info("ListSnapshots()")
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS); err != nil {
		glog.V(3).Infof("invalid list snapshot req: %v", req)
		return nil, err
	}

	return &csi.ListSnapshotsResponse{}, nil
}

//func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
//	glog.V(9).Info("ControllerExpandVolume()")
//	glog.V(9).Info(spew.Sdump(req))
//	return &csi.ControllerExpandVolumeResponse{}, nil
//}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	glog.V(9).Info("GetCapacity()")
	glog.V(9).Info(spew.Sdump(req))
	return &csi.GetCapacityResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	glog.V(9).Info("ControllerPublishVolume()")
	glog.V(9).Info(spew.Sdump(req))
	response := &csi.ControllerPublishVolumeResponse{}
	return response, nil
}