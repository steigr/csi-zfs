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

package main

import (
	"flag"
	"github.com/golang/glog"
	"os"
	"time"

	"github.com/steigr/csi-zfs/pkg/zfs"
)

func init() {
	flag.Set("logtostderr", "true")
}

const (
	backendTimeout = 5 * time.Second
)

var (
	endpoint      = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	driverName    = flag.String("drivername", "zfs", "name of the driver")
	nodeID        = flag.String("nodeid", "", "node id")
	backend       = flag.String("backend", "tcp://127.0.0.1:1737", "zfsd endpoint")
	provisionRoot = flag.String("provision-root", "/var/lib/storage/", "mountpoint for volumes")
)

func main() {
	flag.Parse()

	handle()
	os.Exit(0)
}

func handle() {
	var err error
	driver := zfs.GetDriver()

	driver.ProvisionRoot = *provisionRoot
	if driver.Pools, err = zfs.NewPoolClient(*backend); err != nil {
		glog.Fatal(err)
	}

	if driver.Datasets, err = zfs.NewDatasetClient(*backend); err != nil {
		glog.Fatal(err)
	}
	// TODO: add prometheus handler
	driver.Run(*driverName, *nodeID, *endpoint)
}
